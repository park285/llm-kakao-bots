import { NavLink } from 'react-router-dom';
import { ShieldCheck } from 'lucide-react';
import { routes } from '../utils';

export function Sidebar() {
    return (
        <aside className="sidebar">
            <div className="flex-center gap-2 mb-4" style={{ justifyContent: 'flex-start' }}>
                <ShieldCheck size={32} color="var(--primary)" />
                <span className="text-2xl">Watchdog</span>
            </div>

            <nav style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                {routes.map((route) => (
                    <NavLink
                        key={route.path}
                        to={route.path}
                        className={({ isActive }) =>
                            `flex-center gap-2 ${isActive ? 'active' : ''}`
                        }
                        style={({ isActive }) => ({
                            padding: '0.75rem 1rem',
                            borderRadius: '8px',
                            textDecoration: 'none',
                            color: isActive ? 'white' : 'var(--text-muted)',
                            backgroundColor: isActive ? 'var(--primary)' : 'transparent',
                            justifyContent: 'flex-start',
                            fontWeight: isActive ? 500 : 400,
                            transition: 'all 0.2s'
                        })}
                    >
                        {route.icon && <route.icon size={20} />}
                        {route.label}
                    </NavLink>
                ))}
            </nav>

            <div style={{ marginTop: 'auto' }}>
                <div className="card glass" style={{ padding: '1rem' }}>
                    <div className="text-sm">System Status</div>
                    <div className="text-success flex-center gap-2" style={{ justifyContent: 'flex-start', marginTop: '0.5rem' }}>
                        <div style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: 'var(--success)' }}></div>
                        Operational
                    </div>
                </div>
            </div>
        </aside>
    );
}
