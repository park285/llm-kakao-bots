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
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	defaultUsdKrwRate    = 1400.0
	exchangeRateAPIURL   = "https://api.frankfurter.app/latest?from=USD&to=KRW"
	exchangeRateCacheTTL = time.Hour
)

// ExchangeRateService USD/KRW 환율 조회 인터페이스.
// - Kotlin 구현과 동일하게 실패 시 기본 환율로 fallback 한다.
type ExchangeRateService interface {
	UsdToKrw(ctx context.Context, usdAmount float64) float64
	RateInfo(ctx context.Context) string
}

// FrankfurterExchangeRateService 는 타입이다.
type FrankfurterExchangeRateService struct {
	client  *http.Client
	apiURL  string
	logger  *slog.Logger
	printer *message.Printer

	mu          sync.Mutex
	cachedRate  float64
	cachedUntil time.Time
}

// NewFrankfurterExchangeRateService 는 동작을 수행한다.
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

// UsdToKrw 는 동작을 수행한다.
func (s *FrankfurterExchangeRateService) UsdToKrw(ctx context.Context, usdAmount float64) float64 {
	rate := s.getUsdKrwRate(ctx)
	return usdAmount * rate
}

// RateInfo 는 동작을 수행한다.
func (s *FrankfurterExchangeRateService) RateInfo(ctx context.Context) string {
	rate := s.getUsdKrwRate(ctx)
	rounded := int64(math.Round(rate))
	return fmt.Sprintf("1 USD = %s KRW", s.printer.Sprintf("%d", rounded))
}

func (s *FrankfurterExchangeRateService) getUsdKrwRate(ctx context.Context) float64 {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cachedUntil.IsZero() && now.Before(s.cachedUntil) && s.cachedRate > 0 {
		return s.cachedRate
	}

	rate := s.fetchUsdKrwRate(ctx)
	s.cachedRate = rate
	s.cachedUntil = now.Add(exchangeRateCacheTTL)
	return rate
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
