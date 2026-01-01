import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '@/stores/authStore'
import { CONFIG } from '@/config/constants'
import { getClientHints, getClientHintsHeaders, type ClientHintsData } from '@/utils/clientHints'

// Client Hints 캐시 (앱 시작 시 한 번만 수집)
let clientHintsCache: ClientHintsData | null = null
let clientHintsPromise: Promise<ClientHintsData> | null = null

/**
 * Client Hints를 초기화하고 캐시합니다.
 * 앱 로드 시 한 번만 호출하여 성능을 최적화합니다.
 */
async function ensureClientHints(): Promise<ClientHintsData> {
  if (clientHintsCache) return clientHintsCache
  if (!clientHintsPromise) {
    clientHintsPromise = getClientHints().then(hints => {
      clientHintsCache = hints
      return hints
    })
  }
  return clientHintsPromise
}

// 앱 시작 시 Client Hints 수집 시작 (비동기)
void ensureClientHints()

// API 클라이언트 생성
const apiClient = axios.create({
  baseURL: CONFIG.api.baseUrl,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: CONFIG.api.timeoutMs,
})

// Request interceptor: 민감한 정보 URL 파라미터 방지 + Client Hints 헤더 추가
apiClient.interceptors.request.use(async (config: InternalAxiosRequestConfig) => {
  // 민감한 정보 URL 파라미터 방지
  if (config.params != null && typeof config.params === 'object') {
    const params = config.params as Record<string, unknown>
    delete params['password']
    delete params['token']
  }

  // Client Hints 헤더 추가 (모든 요청에 포함)
  if (clientHintsCache) {
    const hintsHeaders = getClientHintsHeaders(clientHintsCache)
    Object.assign(config.headers, hintsHeaders)
  }

  return config
})

// Response interceptor: \uc5d0\ub7ec \ubc0f \uc778\uc99d \ucc98\ub9ac
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ retry_after?: number }>) => {
    if (axios.isAxiosError(error)) {
      if (error.response?.status === 401) {
        // React 컴포넌트 외부에서 스토어 접근
        useAuthStore.getState().logout()

        // 로그인 페이지로 리다이렉트 (이미 로그인 페이지가 아닌 경우)
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
      } else if (error.response?.status === 429) {
        // Rate limit 처리
        const retryAfter = error.response.data.retry_after ??
          (typeof error.response.headers['retry-after'] === 'string'
            ? parseInt(error.response.headers['retry-after'], 10)
            : undefined)
        console.warn(`Rate limited. Retry after ${String(retryAfter)}s`)
      }
    }
    return Promise.reject(error)
  }
)

export default apiClient
