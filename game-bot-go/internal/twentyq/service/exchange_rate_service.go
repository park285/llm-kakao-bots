package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"
	"golang.org/x/sync/singleflight"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	defaultUsdKrwRate           = 1400.0
	exchangeRateAPIURL          = "https://api.frankfurter.app/latest?from=USD&to=KRW"
	exchangeRateCacheTTL        = time.Hour
	exchangeRateSingleflightKey = "exchange_rate_usd_krw"
)

// ExchangeRateService USD/KRW 환율 조회 인터페이스입니다.
// - Kotlin 구현과 동일하게 실패 시 기본 환율로 fallback 합니다.
type ExchangeRateService interface {
	UsdToKrw(ctx context.Context, usdAmount float64) float64
	RateInfo(ctx context.Context) string
}

// FrankfurterExchangeRateService: Frankfurter API를 이용한 환율 서비스 구현체
type FrankfurterExchangeRateService struct {
	client  *http.Client
	apiURL  string
	logger  *slog.Logger
	printer *message.Printer

	mu          sync.RWMutex
	sf          singleflight.Group
	cachedRate  float64
	cachedUntil time.Time
}

// NewFrankfurterExchangeRateService: 새로운 FrankfurterExchangeRateService 인스턴스를 생성합니다.
func NewFrankfurterExchangeRateService(logger *slog.Logger) *FrankfurterExchangeRateService {
	if logger == nil {
		logger = slog.Default()
	}
	return &FrankfurterExchangeRateService{
		client:  &http.Client{Timeout: 10 * time.Second},
		apiURL:  exchangeRateAPIURL,
		logger:  logger,
		printer: message.NewPrinter(language.Korean),
	}
}

// UsdToKrw: USD 금액을 KRW로 변환합니다.
func (s *FrankfurterExchangeRateService) UsdToKrw(ctx context.Context, usdAmount float64) float64 {
	rate := s.getUsdKrwRate(ctx)
	return usdAmount * rate
}

// RateInfo: 현재 적용 중인 환율 정보를 문자열로 반환합니다.
func (s *FrankfurterExchangeRateService) RateInfo(ctx context.Context) string {
	rate := s.getUsdKrwRate(ctx)
	rounded := int64(math.Round(rate))
	return fmt.Sprintf("1 USD = %s KRW", s.printer.Sprintf("%d", rounded))
}

func (s *FrankfurterExchangeRateService) getUsdKrwRate(ctx context.Context) float64 {
	now := time.Now()

	s.mu.RLock()
	cachedRate := s.cachedRate
	cachedUntil := s.cachedUntil
	s.mu.RUnlock()

	if !cachedUntil.IsZero() && now.Before(cachedUntil) && cachedRate > 0 {
		return cachedRate
	}

	value, _, _ := s.sf.Do(exchangeRateSingleflightKey, func() (any, error) {
		now := time.Now()

		s.mu.RLock()
		cachedRate := s.cachedRate
		cachedUntil := s.cachedUntil
		s.mu.RUnlock()

		if !cachedUntil.IsZero() && now.Before(cachedUntil) && cachedRate > 0 {
			return cachedRate, nil
		}

		rate := s.fetchUsdKrwRate(ctx)
		s.mu.Lock()
		s.cachedRate = rate
		s.cachedUntil = time.Now().Add(exchangeRateCacheTTL)
		s.mu.Unlock()
		return rate, nil
	})

	if rate, ok := value.(float64); ok {
		return rate
	}
	return defaultUsdKrwRate
}

type frankfurterResponse struct {
	Rates map[string]float64 `json:"rates"`
}

func (s *FrankfurterExchangeRateService) fetchUsdKrwRate(ctx context.Context) float64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiURL, http.NoBody)
	if err != nil {
		s.logger.Warn("exchange_rate_request_build_failed", "err", err)
		return defaultUsdKrwRate
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("exchange_rate_request_failed", "err", err)
		return defaultUsdKrwRate
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		s.logger.Warn("exchange_rate_read_failed", "err", err)
		return defaultUsdKrwRate
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warn("exchange_rate_http_failed", "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		return defaultUsdKrwRate
	}

	var parsed frankfurterResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.logger.Warn("exchange_rate_unmarshal_failed", "err", err)
		return defaultUsdKrwRate
	}

	rate, ok := parsed.Rates["KRW"]
	if !ok || rate <= 0 {
		s.logger.Warn("exchange_rate_parse_failed")
		return defaultUsdKrwRate
	}

	s.logger.Info("exchange_rate_fetched", "rate", rate)
	return rate
}
