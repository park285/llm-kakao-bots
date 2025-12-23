import axios, { AxiosError } from 'axios'
import { useAuthStore } from '../stores/authStore'

// API 클라이언트 생성
const apiClient = axios.create({
  baseURL: '/admin/api',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30초 타임아웃
})

// Request interceptor: 민감한 정보 URL 파라미터 방지
apiClient.interceptors.request.use((config) => {
  if (config.params != null && typeof config.params === 'object') {
    const params = config.params as Record<string, unknown>
    delete params['password']
    delete params['token']
  }
  return config
})

// Response interceptor
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
