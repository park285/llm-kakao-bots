import { useState } from 'react'
import clsx from 'clsx'
import { LayoutList, Network, Activity } from 'lucide-react'
import { TraceExplorer } from '@/components/traces/TraceExplorer'
import { DependencyGraph } from '@/components/traces/DependencyGraph'
import { ServiceMetrics } from '@/components/traces/ServiceMetrics'

// TracesTab: 분산 추적 시스템(Jaeger)의 메인 탭 컨테이너입니다.
// 트레이스 탐색, 의존성 그래프, 서비스 메트릭 뷰를 탭으로 제공하여 통합적인 관측성을 지원합니다.
const TracesTab = () => {
    const [activeTab, setActiveTab] = useState<'traces' | 'dependencies' | 'metrics'>('traces')

    return (
        <div className="space-y-4 h-[calc(100vh-140px)] flex flex-col">
            {/* Header / Tabs */}
            <div className="flex items-center gap-1 border-b border-slate-200 shrink-0">
                <TabButton
                    active={activeTab === 'traces'}
                    onClick={() => setActiveTab('traces')}
                    icon={LayoutList}
                    label="Traces"
                />
                <TabButton
                    active={activeTab === 'dependencies'}
                    onClick={() => setActiveTab('dependencies')}
                    icon={Network}
                    label="Dependencies"
                />
                <TabButton
                    active={activeTab === 'metrics'}
                    onClick={() => setActiveTab('metrics')}
                    icon={Activity}
                    label="Service Metrics"
                />
            </div>

            {/* Content Area */}
            <div className="flex-1 min-h-0 pt-2">
                {activeTab === 'traces' && <TraceExplorer />}
                {activeTab === 'dependencies' && <DependencyGraph />}
                {activeTab === 'metrics' && <ServiceMetrics />}
            </div>
        </div>
    )
}

interface TabButtonProps {
    active: boolean
    onClick: () => void
    icon: React.ElementType
    label: string
}

const TabButton = ({ active, onClick, icon: Icon, label }: TabButtonProps) => (
    <button
        onClick={onClick}
        className={clsx(
            'flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
            active
                ? 'border-indigo-500 text-indigo-600'
                : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
        )}
    >
        <Icon size={16} />
        {label}
    </button>
)

export default TracesTab
