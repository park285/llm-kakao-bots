import { lazy, Suspense } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useAuthStore } from './stores/authStore'
import { Loader2 } from 'lucide-react'

// Eager load (critical path)
import LoginPage from './pages/LoginPage'
import { AppLayout } from './layouts/AppLayout'
import ErrorPage from './components/ErrorPage'

// Lazy load (code splitting)
const StatsTab = lazy(() => import('./components/StatsTab'))
const MembersTab = lazy(() => import('./components/MembersTab'))
const AlarmsTab = lazy(() => import('./components/AlarmsTab'))
const RoomsTab = lazy(() => import('./components/RoomsTab'))
const StreamsTab = lazy(() => import('./components/StreamsTab'))
const LogsTab = lazy(() => import('./components/LogsTab'))
const SettingsTab = lazy(() => import('./components/SettingsTab'))

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

// Protected Route Shield
const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)

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
