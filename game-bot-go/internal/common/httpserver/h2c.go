package httpserver

import (
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// WrapH2C: 표준 HTTP 핸들러를 H2C(HTTP/2 Cleartext)를 지원하도록 래핑합니다.
func WrapH2C(handler http.Handler) http.Handler {
	return h2c.NewHandler(handler, &http2.Server{})
}
