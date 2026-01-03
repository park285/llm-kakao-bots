package config

import (
	"os"
	"strconv"
	"strings"
)

func parseAPIKeys() []string {
	keysValue := strings.TrimSpace(os.Getenv("GOOGLE_API_KEYS"))
	if keysValue != "" {
		return splitKeys(keysValue)
	}
	key := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	if key == "" {
		return nil
	}
	return []string{key}
}

func splitKeys(value string) []string {
	items := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isGemini3(model string) bool {
	return strings.Contains(strings.ToLower(model), "gemini-3")
}

func getEnvString(key string, def string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	return value
}

func getEnvInt(key string, def int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvNonNegativeInt(key string, def int) int {
	value := getEnvInt(key, def)
	if value < 0 {
		return 0
	}
	return value
}

func getEnvFloat(key string, def float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvBool(key string, def bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

func maskSecret(value string) string {
	if value == "" {
		return "<missing>"
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + "***" + value[len(value)-2:]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readTelemetryConfig: OpenTelemetry 설정을 환경 변수에서 읽습니다.
func readTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		Enabled:        getEnvBool("OTEL_ENABLED", false),
		ServiceName:    getEnvString("OTEL_SERVICE_NAME", "mcp-llm-server"),
		ServiceVersion: getEnvString("OTEL_SERVICE_VERSION", "1.0.0"),
		Environment:    getEnvString("OTEL_ENVIRONMENT", "production"),
		OTLPEndpoint:   getEnvString("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317"),
		OTLPInsecure:   getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		SampleRate:     getEnvFloat("OTEL_SAMPLE_RATE", 1.0),
	}
}
