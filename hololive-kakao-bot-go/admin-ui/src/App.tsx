import { lazy, Suspense, useEffect, useRef, useCallback } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/authStore'
import { authApi } from '@/api'
import { Loader2 } from 'lucide-react'

// Eager load (critical path)
import LoginPage from '@/pages/LoginPage'
import { AppLayout } from '@/layouts/AppLayout'
import ErrorPage from '@/components/ErrorPage'

// Lazy load (code splitting)
const StatsTab = lazy(() => import('@/components/StatsTab'))
const MembersTab = lazy(() => import('@/components/MembersTab'))
const AlarmsTab = lazy(() => import('@/components/AlarmsTab'))
const RoomsTab = lazy(() => import('@/components/RoomsTab'))
const StreamsTab = lazy(() => import('@/components/StreamsTab'))
const LogsTab = lazy(() => import('@/components/LogsTab'))
const SettingsTab = lazy(() => import('@/components/SettingsTab'))

// Loading Fallback Component
const TabLoader = () => (
  <div className="flex items-center justify-center h-64 text-slate-400">
    <Loader2 className="w-6 h-6 animate-spin mr-2" />
    <span className="text-sm font-medium">로딩 중...</span>
  </div>
)

// QueryClient Configuration
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 60, // 1 hour
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
})

// Heartbeat 설정: 15분 간격, 3회 연속 실패 시 로그아웃
const HEARTBEAT_INTERVAL_MS = 15 * 60 * 1000
const MAX_HEARTBEAT_FAILURES = 3

// Protected Route Shield (with session heartbeat)
const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const logout = useAuthStore((state) => state.logout)
  const intervalRef = useRef<number | null>(null)
  const failCountRef = useRef(0)

  const sendHeartbeat = useCallback(async () => {
    const success = await authApi.heartbeat()
    if (success) {
      // 성공 시 실패 카운터 초기화
      failCountRef.current = 0
    } else {
      // 실패 시 카운터 증가
      failCountRef.current += 1
      console.warn(`Heartbeat failed (${String(failCountRef.current)}/${String(MAX_HEARTBEAT_FAILURES)})`)

      if (failCountRef.current >= MAX_HEARTBEAT_FAILURES) {
        // 3회 연속 실패 → 세션 만료로 판단하여 로그아웃
        logout()
      }
    }
  }, [logout])

  useEffect(() => {
    if (!isAuthenticated) return

    // 즉시 한 번 heartbeat 전송 (페이지 로드 시)
    void sendHeartbeat()

    // 15분마다 heartbeat 전송
    intervalRef.current = window.setInterval(() => {
      void sendHeartbeat()
    }, HEARTBEAT_INTERVAL_MS)

    return () => {
      if (intervalRef.current !== null) {
        window.clearInterval(intervalRef.current)
      }
      failCountRef.current = 0 // cleanup 시 초기화
    }
  }, [isAuthenticated, sendHeartbeat])

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

// Lazy Route Wrapper
const LazyRoute = ({ children }: { children: React.ReactNode }) => (
  <Suspense fallback={<TabLoader />}>
    {children}
  </Suspense>
)

// Modern Data Router Configuration ("React Shell" pattern)
const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
    errorElement: <ErrorPage />,
  },
  {
    path: "/dashboard",
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    errorElement: <ErrorPage />,
    children: [
      {
        index: true,
        element: <Navigate to="stats" replace />
      },
      {
        path: "stats",
        element: <LazyRoute><StatsTab /></LazyRoute>
      },
      {
        path: "members",
        element: <LazyRoute><MembersTab /></LazyRoute>
      },
      {
        path: "alarms",
        element: <LazyRoute><AlarmsTab /></LazyRoute>
      },
      {
        path: "rooms",
        element: <LazyRoute><RoomsTab /></LazyRoute>
      },
      {
        path: "streams",
        element: <LazyRoute><StreamsTab /></LazyRoute>
      },
      {
        path: "logs",
        element: <LazyRoute><LogsTab /></LazyRoute>
      },
      {
        path: "settings",
        element: <LazyRoute><SettingsTab /></LazyRoute>
      },
    ]
  },
  {
    path: "/",
    element: <Navigate to="/dashboard" replace />,
    errorElement: <ErrorPage />,
  },
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />,
  }
])

const App = () => (
  <QueryClientProvider client={queryClient}>
    <RouterProvider router={router} />
  </QueryClientProvider>
)

export default App
