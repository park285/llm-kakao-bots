package admin

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const cfAccessJWTHeader = "Cf-Access-Jwt-Assertion"

type cfAccessClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type cfAccessVerifier struct {
	certsURL     string
	expectedIss  string
	expectedAUD  string
	allowedEmail map[string]struct{}

	cacheTTL   time.Duration
	httpClient *http.Client
	logger     *slog.Logger

	mu        sync.RWMutex
	keysByKID map[string]crypto.PublicKey
	expiresAt time.Time
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	Alg string   `json:"alg"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

func (k jwkKey) publicKey() (crypto.PublicKey, error) {
	if len(k.X5c) > 0 {
		der, err := base64.StdEncoding.DecodeString(k.X5c[0])
		if err != nil {
			return nil, fmt.Errorf("x5c decode failed: %w", err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("x5c parse failed: %w", err)
		}
		return cert.PublicKey, nil
	}

	if strings.ToUpper(k.Kty) != "RSA" {
		return nil, fmt.Errorf("unsupported kty: %s", k.Kty)
	}
	if k.N == "" || k.E == "" {
		return nil, errors.New("missing n/e")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("n decode failed: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("e decode failed: %w", err)
	}

	if len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, fmt.Errorf("invalid e length: %d", len(eBytes))
	}
	var eInt uint64
	buf := make([]byte, 8)
	copy(buf[8-len(eBytes):], eBytes)
	eInt = binary.BigEndian.Uint64(buf)
	if eInt > uint64(^uint(0)) {
		return nil, fmt.Errorf("e overflow: %d", eInt)
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(eInt),
	}
	return pub, nil
}

func newCFAccessVerifier(adminCfg Config, logger *slog.Logger) (*cfAccessVerifier, error) {
	teamDomain := normalizeCFAccessTeamDomain(adminCfg.CFAccessTeamDomain)
	if teamDomain == "" {
		return nil, fmt.Errorf("cf access team domain is empty")
	}
	if strings.TrimSpace(adminCfg.CFAccessAUD) == "" {
		return nil, fmt.Errorf("cf access aud is empty")
	}

	allowed := make(map[string]struct{}, len(adminCfg.AllowedEmails))
	for _, email := range adminCfg.AllowedEmails {
		if email == "" {
			continue
		}
		allowed[email] = struct{}{}
	}

	v := &cfAccessVerifier{
		certsURL:     fmt.Sprintf("https://%s/cdn-cgi/access/certs", teamDomain),
		expectedIss:  fmt.Sprintf("https://%s", teamDomain),
		expectedAUD:  adminCfg.CFAccessAUD,
		allowedEmail: allowed,
		cacheTTL:     10 * time.Minute,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:    logger,
		keysByKID: make(map[string]crypto.PublicKey),
		expiresAt: time.Time{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *cfAccessVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.certsURL, nil)
	if err != nil {
		return fmt.Errorf("create certs request failed: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch certs failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioReadAllLimit(resp.Body, 64*1024)
		return fmt.Errorf("fetch certs failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	raw, err := ioReadAllLimit(resp.Body, 2*1024*1024)
	if err != nil {
		return fmt.Errorf("read certs failed: %w", err)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(raw, &jwks); err != nil {
		return fmt.Errorf("jwks unmarshal failed: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return errors.New("jwks keys is empty")
	}

	keys := make(map[string]crypto.PublicKey, len(jwks.Keys))
	for _, key := range jwks.Keys {
		if key.Kid == "" {
			continue
		}
		pub, err := key.publicKey()
		if err != nil {
			v.logger.Warn("jwks_key_parse_failed", "kid", key.Kid, "err", err)
			continue
		}
		keys[key.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks usable keys is empty")
	}

	v.mu.Lock()
	v.keysByKID = keys
	v.expiresAt = time.Now().Add(v.cacheTTL)
	v.mu.Unlock()

	v.logger.Info("cf_access_jwks_refreshed", "keys", len(keys), "expires_in", v.cacheTTL)
	return nil
}

func (v *cfAccessVerifier) ensureFreshKeys(ctx context.Context) error {
	v.mu.RLock()
	needsRefresh := len(v.keysByKID) == 0 || time.Now().After(v.expiresAt)
	v.mu.RUnlock()

	if !needsRefresh {
		return nil
	}
	return v.refreshKeys(ctx)
}

func (v *cfAccessVerifier) keyFunc(token *jwt.Token) (any, error) {
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, errors.New("missing kid")
	}

	v.mu.RLock()
	key, ok := v.keysByKID[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown kid: %s", kid)
	}
	return key, nil
}

func (v *cfAccessVerifier) parseAndValidate(tokenString string) (*cfAccessClaims, error) {
	claims := &cfAccessClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(v.expectedAUD),
		jwt.WithIssuer(v.expectedIss),
	)
	_, err := parser.ParseWithClaims(tokenString, claims, v.keyFunc)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(claims.Email) == "" {
		return nil, errors.New("email claim is empty")
	}
	return claims, nil
}

func (v *cfAccessVerifier) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := strings.TrimSpace(c.GetHeader(cfAccessJWTHeader))
		if tokenString == "" {
			writeAPIError(c, http.StatusUnauthorized, "missing_token", "Cf-Access-Jwt-Assertion 헤더가 필요합니다.")
			return
		}

		refreshCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := v.ensureFreshKeys(refreshCtx); err != nil {
			v.logger.Error("cf_access_jwks_refresh_failed", "err", err)
			writeAPIError(c, http.StatusServiceUnavailable, "jwks_refresh_failed", "인증 키 로딩에 실패했습니다.")
			return
		}

		claims, err := v.parseAndValidate(tokenString)
		if err != nil {
			// 키 회전 등으로 kid가 바뀐 경우를 고려해 1회 갱신 후 재시도합니다.
			if refreshErr := v.refreshKeys(refreshCtx); refreshErr == nil {
				claims, err = v.parseAndValidate(tokenString)
			}
		}
		if err != nil {
			writeAPIError(c, http.StatusUnauthorized, "invalid_token", "유효하지 않은 토큰입니다.")
			return
		}

		if len(v.allowedEmail) > 0 {
			if _, ok := v.allowedEmail[claims.Email]; !ok {
				writeAPIError(c, http.StatusForbidden, "forbidden", "접근이 허용되지 않은 사용자입니다.")
				return
			}
		}

		c.Set("admin_email", claims.Email)
		c.Next()
	}
}
