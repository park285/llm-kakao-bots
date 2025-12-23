import { useToastStore } from '@/stores/toastStore';
import { AnimatePresence, motion } from 'framer-motion';
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react';
import { cn } from '@/utils';

export function ToastContainer() {
    const { toasts, removeToast } = useToastStore();

    return (
        <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 p-4 max-w-sm w-full pointer-events-none">
            <AnimatePresence>
                {toasts.map((toast) => (
                    <motion.div
                        key={toast.id}
                        initial={{ opacity: 0, x: 20, scale: 0.9 }}
                        animate={{ opacity: 1, x: 0, scale: 1 }}
                        exit={{ opacity: 0, x: 20, scale: 0.9 }}
                        className={cn(
                            "pointer-events-auto flex items-start gap-3 p-4 rounded-xl shadow-lg border backdrop-blur-md",
                            toast.type === 'success' ? "bg-emerald-50/90 border-emerald-200 text-emerald-800" :
                                toast.type === 'error' ? "bg-rose-50/90 border-rose-200 text-rose-800" :
                                    "bg-sky-50/90 border-sky-200 text-sky-800"
                        )}
                    >
                        {toast.type === 'success' && <CheckCircle size={20} className="shrink-0 text-emerald-500 mt-0.5" />}
                        {toast.type === 'error' && <AlertCircle size={20} className="shrink-0 text-rose-500 mt-0.5" />}
                        {toast.type === 'info' && <Info size={20} className="shrink-0 text-sky-500 mt-0.5" />}

                        <p className="text-sm font-medium leading-relaxed flex-1">{toast.message}</p>

                        <button
                            onClick={() => removeToast(toast.id)}
                            className="text-current opacity-50 hover:opacity-100 transition-opacity p-0.5 -mt-1 -mr-1"
                        >
                            <X size={16} />
                        </button>
                    </motion.div>
                ))}
            </AnimatePresence>
        </div>
    );
}
