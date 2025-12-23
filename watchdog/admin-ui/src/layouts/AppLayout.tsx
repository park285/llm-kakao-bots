import { Outlet, NavLink, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
    LayoutDashboard,
    Server,
    Activity,
    Settings,
    ShieldCheck,
    Menu,
    X,
    LogOut
} from 'lucide-react';
import { useState } from 'react';
import { cn } from '../utils';

export function AppLayout() {
    const location = useLocation();
    const [isSidebarOpen, setIsSidebarOpen] = useState(true);

    const navItems = [
        { id: 'overview', label: 'Overview', icon: LayoutDashboard, path: '/' },
        { id: 'containers', label: 'Containers', icon: Server, path: '/containers' },
        { id: 'events', label: 'Events', icon: Activity, path: '/events' },
        { id: 'settings', label: 'Settings', icon: Settings, path: '/settings' },
    ];

    return (
        <div className="flex h-screen bg-slate-50 overflow-hidden font-display selection:bg-sky-200">
            {/* Dynamic Background */}
            <div className="absolute inset-0 z-0 pointer-events-none">
                <div className="absolute top-0 left-0 w-full h-96 bg-gradient-to-b from-sky-50/50 to-transparent"></div>
            </div>

            {/* Sidebar */}
            <motion.aside
                initial={false}
                animate={{ width: isSidebarOpen ? 260 : 80 }}
                className="bg-white/80 backdrop-blur-xl border-r border-slate-200 z-20 flex flex-col transition-all duration-300 relative shadow-sm"
            >
                {/* Logo Area */}
                <div className="h-20 flex items-center justify-between px-6 border-b border-slate-100">
                    <AnimatePresence mode="wait">
                        {isSidebarOpen ? (
                            <motion.div
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                exit={{ opacity: 0 }}
                                className="flex items-center gap-3"
                            >
                                <div className="w-8 h-8 bg-gradient-to-br from-indigo-500 to-purple-500 rounded-lg flex items-center justify-center shadow-md shadow-indigo-200">
                                    <ShieldCheck className="w-5 h-5 text-white" />
                                </div>
                                <span className="text-lg font-bold text-slate-800 tracking-tight">
                                    Watchdog
                                </span>
                            </motion.div>
                        ) : (
                            <motion.div
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                className="mx-auto w-8 h-8 bg-gradient-to-br from-indigo-500 to-purple-500 rounded-lg flex items-center justify-center shadow-md shadow-indigo-200"
                            >
                                <ShieldCheck className="w-5 h-5 text-white" />
                            </motion.div>
                        )}
                    </AnimatePresence>
                    {isSidebarOpen && (
                        <button
                            onClick={() => setIsSidebarOpen(false)}
                            className="p-1.5 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
                        >
                            <X size={18} />
                        </button>
                    )}
                </div>

                {!isSidebarOpen && (
                    <div className="py-4 flex justify-center border-b border-slate-100">
                        <button
                            onClick={() => setIsSidebarOpen(true)}
                            className="p-1.5 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
                        >
                            <Menu size={20} />
                        </button>
                    </div>
                )}

                {/* Navigation */}
                <nav className="flex-1 py-6 px-3 space-y-1.5 overflow-y-auto scrollbar-hide">
                    {navItems.map((item) => (
                        <NavLink
                            key={item.id}
                            to={item.path}
                            className={({ isActive }) => cn(
                                "flex items-center px-3 py-3.5 rounded-xl transition-all duration-200 group relative overflow-hidden",
                                isActive
                                    ? "bg-sky-50 text-sky-600 shadow-sm shadow-sky-100"
                                    : "text-slate-500 hover:bg-slate-50 hover:text-slate-900"
                            )}
                        >
                            {({ isActive }) => (
                                <>
                                    <item.icon
                                        size={22}
                                        strokeWidth={isActive ? 2.5 : 2}
                                        className={cn("shrink-0 transition-colors", isActive ? "text-sky-500" : "text-slate-400 group-hover:text-slate-600")}
                                    />
                                    {isSidebarOpen && (
                                        <motion.span
                                            initial={{ opacity: 0, x: -10 }}
                                            animate={{ opacity: 1, x: 0 }}
                                            className="ml-3 font-medium whitespace-nowrap"
                                        >
                                            {item.label}
                                        </motion.span>
                                    )}
                                    {isActive && (
                                        <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-sky-500 rounded-r-full" />
                                    )}
                                </>
                            )}
                        </NavLink>
                    ))}
                </nav>

                {/* Footer / Logout Placeholder */}
                <div className="p-4 border-t border-slate-100">
                    <button
                        className={cn(
                            "flex items-center w-full p-3.5 rounded-xl hover:bg-rose-50 text-slate-500 hover:text-rose-600 transition-colors group cursor-pointer",
                            !isSidebarOpen && "justify-center"
                        )}
                        onClick={() => alert('Logout clicked')}
                    >
                        <LogOut size={20} className="group-hover:stroke-rose-600 transition-colors" />
                        {isSidebarOpen && <span className="ml-3 font-medium">Logout</span>}
                    </button>
                </div>
            </motion.aside>

            {/* Main Content */}
            <main className="flex-1 flex flex-col min-w-0 overflow-hidden relative z-10">
                {/* Header */}
                <header className="h-20 bg-white/60 backdrop-blur-md border-b border-slate-200/50 flex items-center justify-between px-8 sticky top-0 z-20">
                    <div>
                        <h2 className="text-2xl font-bold text-slate-800 tracking-tight">
                            {navItems.find(i => i.path === location.pathname)?.label || 'Dashboard'}
                        </h2>
                        <p className="text-xs text-slate-400 font-medium mt-0.5">
                            Watchdog Control Center
                        </p>
                    </div>

                    <div className="flex items-center space-x-4">
                        <div className="flex items-center space-x-3 px-1 py-1 bg-white border border-slate-200 rounded-full shadow-sm pr-4">
                            <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-indigo-500 to-purple-500 flex items-center justify-center text-white font-bold text-sm shadow-sm ring-2 ring-white">
                                A
                            </div>
                            <div className="flex flex-col">
                                <span className="text-sm font-bold text-slate-700 leading-none">Admin</span>
                                <span className="text-[10px] text-indigo-500 font-medium leading-none mt-1">System</span>
                            </div>
                        </div>
                    </div>
                </header>

                <div className="flex-1 overflow-auto p-6 sm:p-10 scroll-smooth">
                    <div className="max-w-7xl mx-auto w-full">
                        <Outlet />
                    </div>
                </div>
            </main>
        </div>
    );
}
