import { useRouteError, isRouteErrorResponse } from 'react-router-dom';
import { RefreshCw, AlertTriangle, Home } from 'lucide-react';
import { extractErrorMessage } from '@/lib/typeUtils';

const ErrorPage = () => {
    const error = useRouteError();
    console.error(error); // 디버깅용 로그

    let errorMessage = '예기치 않은 오류가 발생했습니다.';
    let errorTitle = '시스템 오류';

    if (isRouteErrorResponse(error)) {
        errorMessage = error.statusText || extractErrorMessage(error.data) || '페이지를 찾을 수 없습니다.';
        if (error.status === 404) {
            errorTitle = 'Page Not Found';
            errorMessage = '요청하신 페이지가 존재하지 않거나 이동되었습니다.';
        } else if (error.status === 401 || error.status === 403) {
            errorTitle = '접근 권한 없음';
            errorMessage = '이 페이지에 접근할 권한이 없습니다. 다시 로그인 해주세요.';
        }
    } else if (error instanceof Error) {
        errorMessage = error.message;
    } else if (typeof error === 'string') {
        errorMessage = error;
    }

    const handleReload = () => { window.location.reload(); };
    const handleGoHome = () => { window.location.href = '/dashboard'; };

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-50 p-6 font-sans">
            <div className="max-w-md w-full bg-white border border-slate-200 rounded-2xl shadow-xl shadow-slate-200/50 p-8 text-center animate-in fade-in zoom-in-95 duration-300">

                {/* 아이콘 Wrapper (gradient ring 효과) */}
                <div className="mb-6 flex justify-center">
                    <div className="relative">
                        <div className="absolute inset-0 bg-rose-100 rounded-full blur-xl opacity-70 animate-pulse"></div>
                        <div className="relative p-4 bg-white rounded-full border border-rose-100 shadow-sm">
                            <AlertTriangle className="w-10 h-10 text-rose-500" strokeWidth={2} />
                        </div>
                    </div>
                </div>

                <h1 className="text-2xl font-bold text-slate-800 mb-2 tracking-tight">
                    {errorTitle}
                </h1>

                <p className="text-slate-500 mb-8 leading-relaxed text-sm px-4">
                    {errorMessage}
                </p>

                <div className="flex flex-col sm:flex-row gap-3 justify-center">
                    <button
                        onClick={handleReload}
                        className="flex items-center justify-center gap-2 px-5 py-2.5 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 font-semibold rounded-xl transition-all duration-200 shadow-sm active:scale-95 text-sm"
                    >
                        <RefreshCw className="w-4 h-4" />
                        새로고침
                    </button>

                    <button
                        onClick={handleGoHome}
                        className="flex items-center justify-center gap-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold rounded-xl transition-all duration-200 shadow-lg shadow-indigo-500/20 active:scale-95 text-sm"
                    >
                        <Home className="w-4 h-4" />
                        대시보드로 이동
                    </button>
                </div>

                {/* 개발 모드 Stack Trace */}
                {import.meta.env.DEV && error instanceof Error && (
                    <div className="mt-8 text-left">
                        <div className="bg-slate-900 rounded-lg p-4 overflow-auto max-h-48 border border-slate-800 shadow-inner">
                            <code className="text-[10px] font-mono text-rose-300 break-all whitespace-pre-wrap leading-tight block">
                                <span className="text-slate-500 block mb-2 border-b border-slate-700 pb-1">Debug Info:</span>
                                {error.stack}
                            </code>
                        </div>
                    </div>
                )}
            </div>

            {/* 배경 장식 */}
            <div className="fixed top-0 left-0 w-full h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500" />
            <div className="fixed bottom-4 text-center w-full text-xs text-slate-400">
                Hololive Kakao Bot Admin
            </div>
        </div>
    );
};

export default ErrorPage;
