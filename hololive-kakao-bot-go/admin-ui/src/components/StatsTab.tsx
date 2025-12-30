import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { statsApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import { StatCard } from '@/components/ui'
import { Users, Bell, MessageSquare, Loader2, ArrowRight, Activity, Server, ShieldCheck, Bot } from 'lucide-react'
import { motion } from 'framer-motion'
import { SystemStatsChart, ChannelStatsTable } from '@/components/dashboard'

const StatsTab = () => {
  const navigate = useNavigate()
  const { data: response, isLoading } = useQuery({
    queryKey: queryKeys.stats.summary,
    queryFn: statsApi.get,
    refetchInterval: 10000,
  })

  // 바로가기 핸들러
  const go = (path: string) => { void navigate(path) }

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64 text-slate-400">
        <Loader2 className="animate-spin mr-2" />
        데이터를 불러오는 중...
      </div>
    )
  }

  const mainStats = [
    {
      label: '등록된 멤버',
      value: response?.members || 0,
      variant: 'cyan' as const,
      icon: <Users size={24} />,
    },
    {
      label: '활성 알람',
      value: response?.alarms || 0,
      variant: 'rose' as const,
      icon: <Bell size={24} />,
    },
    {
      label: '연동된 방',
      value: response?.rooms || 0,
      variant: 'indigo' as const,
      icon: <MessageSquare size={24} />,
    },
  ]

  return (
    <div className="space-y-8">
      {/* 1. 환영 배너 */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative overflow-hidden rounded-3xl bg-white border border-slate-100 p-8 shadow-sm"
      >
        {/* 배경 Gradients */}
        <div className="absolute top-0 right-0 w-96 h-96 bg-sky-50 rounded-full blur-3xl opacity-60 -mr-20 -mt-20 pointer-events-none"></div>
        <div className="absolute bottom-0 left-0 w-64 h-64 bg-cyan-50 rounded-full blur-3xl opacity-40 -ml-10 -mb-10 pointer-events-none"></div>

        <div className="relative z-10 flex flex-col md:flex-row items-center justify-between gap-8">
          <div className="max-w-xl">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-sky-50 border border-sky-100 text-sky-600 text-xs font-semibold mb-4">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-sky-500"></span>
              </span>
              System Operational
            </div>
            <h1 className="text-3xl font-bold text-slate-800 tracking-tight">
              Hololive Bot Console
            </h1>
          </div>

          {/* Hero 일러스트 */}
          <div className="hidden md:flex items-center justify-center w-32 h-32 bg-gradient-to-br from-sky-400 via-cyan-400 to-indigo-400 rounded-3xl shadow-xl shadow-sky-200 transform rotate-6 border-4 border-white">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              className="w-16 h-16 text-white drop-shadow-md"
              stroke="currentColor"
              strokeWidth="0" // fill 사용
            >
              <path d="M8 5v14l11-7z" fill="currentColor" />
            </svg>
          </div>
        </div>
      </motion.div>

      {/* 2. 주요 지표 */}
      <div>
        <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
          <Activity size={20} className="text-sky-500" />
          실시간 현황
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {mainStats.map((stat, idx) => (
            <motion.div
              key={stat.label}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: idx * 0.1 }}
            >
              <StatCard
                label={stat.label}
                value={stat.value}
                icon={stat.icon}
                variant={stat.variant}
              />
            </motion.div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* 좌측 칼럼: 통계 & 시스템 상태 */}
        <div className="lg:col-span-2 space-y-6">
          <SystemStatsChart />

          {/* 3. 시스템 상태 */}
          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm">
            <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
              <Server size={20} className="text-slate-500" />
              시스템 상태
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="p-4 bg-slate-50 rounded-xl border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-xs text-slate-500 font-medium uppercase tracking-wider mb-1">Server Status</div>
                  <div className="flex items-center gap-2">
                    <span className="relative flex h-3 w-3">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
                    </span>
                    <span className="font-bold text-slate-700">Online</span>
                  </div>
                </div>
                <div className="h-10 w-10 bg-white rounded-full flex items-center justify-center border border-slate-200">
                  <ShieldCheck size={20} className="text-emerald-500" />
                </div>
              </div>
              <div className="p-4 bg-slate-50 rounded-xl border border-slate-100 flex items-center justify-between">
                <div>
                  <div className="text-xs text-slate-500 font-medium uppercase tracking-wider mb-1">Bot Version</div>
                  <div className="font-bold text-slate-700 font-mono">{response?.version || 'Unknown'}</div>
                  <div className="text-[10px] text-slate-400 mt-1">Uptime: {response?.uptime || '-'}</div>
                </div>
                <div className="h-10 w-10 bg-white rounded-full flex items-center justify-center border border-slate-200">
                  <Bot size={20} className="text-indigo-500" />
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* 4. 바로가기 */}
        <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm flex flex-col">
          <h3 className="text-lg font-bold text-slate-800 mb-4">바로가기</h3>
          <div className="space-y-3 flex-1">
            <button
              onClick={() => { go('/dashboard/members') }}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-sky-50 text-sky-700 hover:bg-sky-100 transition-colors group text-left"
            >
              <span className="font-medium">멤버 관리하기</span>
              <ArrowRight size={18} className="opacity-50 group-hover:opacity-100 group-hover:translate-x-1 transition-all" />
            </button>
            <button
              onClick={() => { go('/dashboard/alarms') }}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-rose-50 text-rose-700 hover:bg-rose-100 transition-colors group text-left"
            >
              <span className="font-medium">알람 설정 확인</span>
              <ArrowRight size={18} className="opacity-50 group-hover:opacity-100 group-hover:translate-x-1 transition-all" />
            </button>
            <button
              onClick={() => { go('/dashboard/rooms') }}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-indigo-50 text-indigo-700 hover:bg-indigo-100 transition-colors group text-left"
            >
              <span className="font-medium">채팅방 목록</span>
              <ArrowRight size={18} className="opacity-50 group-hover:opacity-100 group-hover:translate-x-1 transition-all" />
            </button>
          </div>
        </div>
      </div>


      {/* 5. 채널 통계 */}
      <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm">
        <h3 className="text-lg font-bold text-slate-800 mb-6 flex items-center gap-2">
          <Activity size={20} className="text-rose-500" />
          채널 통계 (구독자 순 상위 10등)
        </h3>
        <ChannelStatsTable />
      </div>
    </div >
  )
}

export default StatsTab
