import { useRouteError, isRouteErrorResponse } from 'react-router-dom';
import { RefreshCw, AlertTriangle } from 'lucide-react';
import { extractErrorMessage } from '@/lib/typeUtils';

const ErrorPage = () => {
    const error = useRouteError();
    console.error(error); // Log the full error to console for debugging

    // Detailed error checking
    let errorMessage = '알 수 없는 오류가 발생했습니다.';
    let errorTitle = '오류 발생';

    if (isRouteErrorResponse(error)) {
        // Handle standard router errors (404, etc.)
        errorMessage = error.statusText || extractErrorMessage(error.data) || '요청하신 페이지를 찾을 수 없거나 접근할 수 없습니다.';
        if (error.status === 404) {
            errorTitle = '페이지를 찾을 수 없음';
        }
    } else if (error instanceof Error) {
        // Handle generic JS errors
        errorMessage = error.message;
    } else if (typeof error === 'string') {
        errorMessage = error;
    }

    const handleReload = () => {
        window.location.reload();
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-950 p-4">
            <div className="max-w-md w-full bg-slate-900 border border-slate-800 rounded-xl shadow-2xl overflow-hidden p-8 text-center animate-in fade-in zoom-in duration-300">
                <div className="mb-6 flex justify-center">
                    <div className="p-4 bg-red-500/10 rounded-full">
                        <AlertTriangle className="w-12 h-12 text-red-500" />
                    </div>
                </div>

                <h1 className="text-2xl font-bold text-white mb-2 tracking-tight">
                    {errorTitle}
                </h1>

                <p className="text-slate-400 mb-8 leading-relaxed">
                    {errorMessage}
                </p>

                <div className="flex gap-3 justify-center">
                    <button
                        onClick={handleReload}
                        className="flex items-center justify-center px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg transition-all duration-200 shadow-lg shadow-indigo-500/20 active:scale-95"
                    >
                        <RefreshCw className="w-4 h-4 mr-2" />
                        페이지 새로고침
                    </button>

                    <button
                        onClick={() => window.location.href = '/'}
                        className="flex items-center justify-center px-6 py-3 bg-slate-800 hover:bg-slate-700 text-slate-200 font-medium rounded-lg transition-all duration-200 border border-slate-700 active:scale-95"
                    >
                        홈으로 이동
                    </button>
                </div>

                {import.meta.env.DEV && error instanceof Error && (
                    <div className="mt-8 p-4 bg-slate-950 rounded text-left overflow-auto max-h-40 border border-slate-800">
                        <p className="text-xs font-mono text-red-400 break-all">
                            {error.stack}
                        </p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ErrorPage;
