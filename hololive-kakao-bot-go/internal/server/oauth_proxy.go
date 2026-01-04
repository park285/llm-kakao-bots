// OAuth 콜백 프록시
// 모바일 앱에서 localhost 리디렉션이 불가능하기 때문에
// 서버를 통해 OAuth 콜백을 받아 Deep Link로 앱에 전달합니다.

package server

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	// 앱 Deep Link 스키마 (Android에서 등록 필요)
	appScheme = "hololive-app"
	// 콜백 경로
	callbackPath = "callback"
)

// OAuthCallbackHandler: OAuth 프록시 콜백을 처리하여 Deep Link로 리디렉트합니다.
// GET /oauth/callback?code=XXX&state=YYY
// -> hololive-app://callback?code=XXX&state=YYY
func (h *APIHandler) OAuthCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")

	// Deep Link URL 생성
	deepLinkURL := buildDeepLinkURL(code, state, errorParam, errorDesc)

	// HTML 페이지로 리디렉트 (JavaScript 폴백 포함)
	// 일부 브라우저에서 Deep Link 직접 리디렉션이 차단될 수 있으므로 HTML+JS 방식 사용
	htmlResponse := buildRedirectHTML(deepLinkURL, errorParam != "")

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, htmlResponse)
}

// buildDeepLinkURL: Deep Link URL을 생성합니다.
func buildDeepLinkURL(code, state, errorParam, errorDesc string) string {
	baseURL := fmt.Sprintf("%s://%s", appScheme, callbackPath)
	params := url.Values{}

	if errorParam != "" {
		params.Set("error", errorParam)
		if errorDesc != "" {
			params.Set("error_description", errorDesc)
		}
	} else if code != "" {
		params.Set("code", code)
		if state != "" {
			params.Set("state", state)
		}
	}

	if len(params) > 0 {
		return baseURL + "?" + params.Encode()
	}
	return baseURL
}

// buildRedirectHTML: Deep Link로 리디렉트하는 HTML 페이지를 생성합니다.
func buildRedirectHTML(deepLinkURL string, isError bool) string {
	status := "로그인 처리 중..."
	icon := "⏳"
	color := "#667eea"

	if isError {
		status = "로그인 실패"
		icon = "❌"
		color = "#e74c3c"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Hololive App - OAuth</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            background: linear-gradient(135deg, %s 0%%, #764ba2 100%%);
        }
        .container {
            text-align: center;
            color: #fff;
            padding: 2rem;
            max-width: 400px;
        }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { margin-bottom: 16px; font-size: 24px; }
        p { opacity: 0.9; margin-bottom: 24px; line-height: 1.6; }
        .button {
            display: inline-block;
            padding: 12px 32px;
            background: rgba(255,255,255,0.2);
            border: 2px solid rgba(255,255,255,0.5);
            border-radius: 8px;
            color: #fff;
            text-decoration: none;
            font-weight: 600;
            transition: all 0.2s;
        }
        .button:hover {
            background: rgba(255,255,255,0.3);
        }
        .help {
            margin-top: 32px;
            padding-top: 16px;
            border-top: 1px solid rgba(255,255,255,0.2);
            font-size: 14px;
            opacity: 0.7;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">%s</div>
        <h1>%s</h1>
        <p>앱이 자동으로 열리지 않으면 아래 버튼을 눌러주세요.</p>
        <a href="%s" class="button" id="openApp">앱 열기</a>
        <div class="help">
            <p>문제가 계속되면 앱을 다시 설치해주세요.</p>
        </div>
    </div>
    <script>
        // 자동으로 Deep Link 열기 시도
        window.location.href = '%s';
        
        // 3초 후에도 이 페이지가 보이면 수동 버튼 강조
        setTimeout(function() {
            document.getElementById('openApp').style.background = 'rgba(255,255,255,0.4)';
            document.getElementById('openApp').style.transform = 'scale(1.05)';
        }, 3000);
    </script>
</body>
</html>`, color, icon, status, deepLinkURL, deepLinkURL)
}
