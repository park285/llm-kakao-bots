import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AppLayout } from './layouts/AppLayout';
import { Dashboard } from './pages/Dashboard';
import { ContainerDetailPage } from './pages/ContainerDetailPage';
import { DockerInventoryPage } from './pages/DockerInventoryPage';
import { EventsLog } from './pages/EventsLog';
import { SettingsPage } from './pages/SettingsPage';
import { ContainerInfo } from './types';
import { getTargets } from './api/client';
import { useEffect, useState } from 'react';
import { ContainerCard } from './components/ContainerCard';
import { Loader2, Server, AlertTriangle } from 'lucide-react';
import { ToastContainer } from './components/ToastContainer';

// Container List Page (Managed Targets Only)
function ContainersList() {
  const [targets, setTargets] = useState<ContainerInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(function load() {
    async function fetch() {
      try {
        const data = await getTargets();
        setTargets(data.targets || []);
      } catch (err) {
        console.error('Failed to load targets', err);
      } finally {
        setLoading(false);
      }
    }
    fetch();
    const interval = setInterval(fetch, 5000);
    return function cleanup() { clearInterval(interval); };
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400">
        <Loader2 className="animate-spin mr-2" />
        Loading managed containers...
      </div>
    );
  }

  if (targets.length === 0) {
    return (
      <div className="p-8 bg-amber-50 border border-amber-200 rounded-2xl text-center max-w-xl mx-auto">
        <AlertTriangle className="w-10 h-10 text-amber-500 mx-auto mb-3" />
        <h3 className="text-lg font-bold text-slate-800 mb-2">No Managed Containers</h3>
        <p className="text-slate-600 text-sm">
          Configure containers via <code className="bg-slate-200 px-1 rounded">WATCHDOG_CONTAINERS</code> env or config JSON.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-800 flex items-center gap-2">
        <Server size={24} className="text-slate-400" />
        Managed Containers
      </h1>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {targets.map(c => (
          <ContainerCard key={c.id} container={c} />
        ))}
      </div>
    </div>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/containers" element={<ContainersList />} />
          <Route path="/containers/:name" element={<ContainerDetailPage />} />
          <Route path="/docker" element={<DockerInventoryPage />} />
          <Route path="/events" element={<EventsLog />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
      <ToastContainer />
    </BrowserRouter>
  );
}

