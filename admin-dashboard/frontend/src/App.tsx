import { lazy, Suspense, useEffect, useRef, useCallback } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/authStore'
import { authApi } from '@/api'
import { Loader2 } from 'lucide-react'
import toast, { Toaster } from 'react-hot-toast'
import { getErrorMessageFromUnknown } from '@/lib/typeUtils'
import { CONFIG } from '@/config'
import { useActivityDetection } from '@/hooks/useActivityDetection'

// Eager load (핵심 경로 - 즉시 로드)
import LoginPage from '@/pages/LoginPage'
import { AppLayout } from '@/layouts/AppLayout'
import ErrorPage from '@/components/ErrorPage'

// Lazy load (코드 스플리팅)
const StatsTab = lazy(() => import('@/components/StatsTab'))
const MembersTab = lazy(() => import('@/components/MembersTab'))
const MilestonesTab = lazy(() => import('@/components/MilestonesTab'))
const AlarmsTab = lazy(() => import('@/components/AlarmsTab'))
const RoomsTab = lazy(() => import('@/components/RoomsTab'))
const StreamsTab = lazy(() => import('@/components/StreamsTab'))
const LogsTab = lazy(() => import('@/components/LogsTab'))
const TracesTab = lazy(() => import('@/components/TracesTab'))
const SettingsTab = lazy(() => import('@/components/SettingsTab'))
const TwentyQPage = lazy(() => import('@/pages/TwentyQPage'))
const TurtleSoupPage = lazy(() => import('@/pages/TurtleSoupPage'))

// 로딩 Fallback 컴포넌트
const TabLoader = () => (
    <div className="flex items-center justify-center h-64 text-slate-400">
        <Loader2 className="w-6 h-6 animate-spin mr-2" />
        <span className="text-sm font-medium">로딩 중...</span>
    </div>
)

// QueryClient 설정 (글로벌 에러 핸들링 포함)
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
            onError: (error: Error) => {
                // 글로벌 에러 핸들링: 개별 mutation에서 onError를 정의하지 않으면 여기서 처리
                toast.error(getErrorMessageFromUnknown(error))
            },
        },
    },
})

// Heartbeat 설정은 CONFIG에서 관리

// Protected Route (세션 heartbeat + 활동 감지)
const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
    const logout = useAuthStore((state) => state.logout)
    const intervalRef = useRef<number | null>(null)
    const failCountRef = useRef(0)

    // 활동 감지 (설정된 시간 동안 활동 없으면 idle=true)
    const isIdle = useActivityDetection(CONFIG.heartbeat.idleTimeoutMs)

    const sendHeartbeat = useCallback(async (idle: boolean) => {
        try {
            const response = await authApi.heartbeat(idle)

            // 1. 유휴 상태로 갱신 거부됨 (즉시 로그아웃 처리)
            if (response.idle_rejected) {
                console.warn('Session expired due to inactivity')
                logout()
                return
            }

            // 2. 절대 만료 시간 초과
            if (response.absolute_expired) {
                console.warn('Session absolute timeout exceeded')
                toast.error('보안을 위해 세션이 만료되었습니다. 다시 로그인해주세요.')
                logout()
                return
            }

            // 3. 에러 (세션 만료 등)
            if (response.error) {
                if (response.error === 'Session expired') {
                    logout()
                    return
                }
                throw new Error(response.error)
            }

            // 성공 시 실패 카운터 초기화
            failCountRef.current = 0

        } catch (e) {
            failCountRef.current += 1
            console.warn(`Heartbeat failed (${failCountRef.current}/${CONFIG.heartbeat.maxFailures})`)

            if (failCountRef.current >= CONFIG.heartbeat.maxFailures) {
                // 연속 실패 시 로그아웃
                logout()
            }
        }
    }, [logout])

    // 유휴 상태 감지 시 즉시 하트비트 전송 (서버에서 TTL 단축 유도)
    useEffect(() => {
        if (isAuthenticated && isIdle) {
            void sendHeartbeat(true)
        }
    }, [isAuthenticated, isIdle, sendHeartbeat])

    // 페이지 가시성 변경 감지 (탭 복귀 시 즉시 하트비트 check)
    useEffect(() => {
        if (!isAuthenticated) return

        const handleVisibilityChange = () => {
            if (document.visibilityState === 'visible') {
                // 복귀 시 idle=false로 즉시 갱신
                void sendHeartbeat(false)
            }
        }

        document.addEventListener('visibilitychange', handleVisibilityChange)
        return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
    }, [isAuthenticated, sendHeartbeat])

    // 정기 하트비트
    useEffect(() => {
        if (!isAuthenticated) return

        // 초기 실행
        void sendHeartbeat(isIdle)

        // 주기적 실행
        intervalRef.current = window.setInterval(() => {
            void sendHeartbeat(isIdle)
        }, CONFIG.heartbeat.intervalMs)

        return () => {
            if (intervalRef.current !== null) {
                window.clearInterval(intervalRef.current)
            }
            failCountRef.current = 0
        }
    }, [isAuthenticated, isIdle, sendHeartbeat])

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />
    }

    return <>{children}</>
}

// Lazy Route Wrapper 컴포넌트
const LazyRoute = ({ children }: { children: React.ReactNode }) => (
    <Suspense fallback={<TabLoader />}>
        {children}
    </Suspense>
)

// 라우터 설정 (React Shell 패턴)
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
                path: "milestones",
                element: <LazyRoute><MilestonesTab /></LazyRoute>
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
                path: "traces",
                element: <LazyRoute><TracesTab /></LazyRoute>
            },
            {
                path: "settings",
                element: <LazyRoute><SettingsTab /></LazyRoute>
            },
            {
                path: "games/twentyq/*",
                element: <LazyRoute><TwentyQPage /></LazyRoute>
            },
            {
                path: "games/turtlesoup/*",
                element: <LazyRoute><TurtleSoupPage /></LazyRoute>
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

// Toast 스타일 설정 (글로벌)
const toastOptions = {
    className: 'text-sm font-medium',
    style: {
        background: '#ffffff',
        color: '#334155',
        padding: '12px 16px',
        borderRadius: '12px',
        boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
        border: '1px solid #f1f5f9',
    },
    success: {
        iconTheme: { primary: '#0ea5e9', secondary: '#ffffff' },
    },
    error: {
        iconTheme: { primary: '#ef4444', secondary: '#ffffff' },
    },
}

const App = () => (
    <QueryClientProvider client={queryClient}>
        <Toaster position="top-center" reverseOrder={false} toastOptions={toastOptions} />
        <RouterProvider router={router} />
    </QueryClientProvider>
)

export default App
