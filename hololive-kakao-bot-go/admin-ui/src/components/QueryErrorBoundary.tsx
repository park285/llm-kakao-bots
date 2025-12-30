/**
 * Query Error Boundary
 * TanStack Query 에러를 graceful하게 처리하는 Error Boundary
 */

import { Component, type ReactNode, type ErrorInfo } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui'

interface ErrorBoundaryProps {
    children: ReactNode
    fallback?: ReactNode
    onError?: (error: Error, errorInfo: ErrorInfo) => void
}

interface ErrorBoundaryState {
    hasError: boolean
    error: Error | null
}

export class QueryErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props)
        this.state = { hasError: false, error: null }
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error }
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
        console.error('QueryErrorBoundary caught an error:', error, errorInfo)
        this.props.onError?.(error, errorInfo)
    }

    handleRetry = (): void => {
        this.setState({ hasError: false, error: null })
    }

    render(): ReactNode {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback
            }

            return (
                <div className="flex flex-col items-center justify-center min-h-[200px] p-8 bg-rose-50 rounded-xl border border-rose-100">
                    <div className="w-12 h-12 bg-rose-100 rounded-full flex items-center justify-center mb-4">
                        <AlertTriangle className="w-6 h-6 text-rose-500" />
                    </div>
                    <h3 className="text-lg font-bold text-slate-800 mb-2">문제가 발생했습니다</h3>
                    <p className="text-sm text-slate-500 mb-4 text-center max-w-md">
                        {this.state.error?.message ?? '알 수 없는 오류가 발생했습니다.'}
                    </p>
                    <Button
                        onClick={this.handleRetry}
                        variant="outline"
                        className="gap-2"
                    >
                        <RefreshCw size={16} />
                        다시 시도
                    </Button>
                </div>
            )
        }

        return this.props.children
    }
}

/**
 * Suspense와 함께 사용하는 간단한 로딩 Fallback
 */
export const QueryLoadingFallback = () => (
    <div className="flex items-center justify-center min-h-[200px] text-slate-400">
        <RefreshCw className="w-5 h-5 animate-spin mr-2" />
        <span className="text-sm font-medium">로딩 중...</span>
    </div>
)
