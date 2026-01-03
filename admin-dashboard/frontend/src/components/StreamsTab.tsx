import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { streamsApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import { ExternalLink, Calendar, PlayCircle } from 'lucide-react'
import type { Stream } from '@/types'

/**
 * 이미지 최적화 헬퍼 (wsrv.nl 오픈 소스 이미지 프록시 사용)
 * - 캐싱, 리사이징, WebP 변환, 압축
 * - 고화질 디스플레이 대응을 위한 2x srcset 제공
 */
const getOptimizedThumbnail = (url?: string, width = 640) => {
    if (!url) return undefined
    // 품질 90%, 너비 640px으로 고화질 유지
    return `https://wsrv.nl/?url=${encodeURIComponent(url)}&w=${String(width)}&q=90&output=webp`
}

// Retina (2x) 대응 srcset 생성
const getThumbnailSrcSet = (url?: string) => {
    if (!url) return undefined
    const w1x = getOptimizedThumbnail(url, 640)
    const w2x = getOptimizedThumbnail(url, 1280)
    return `${w1x} 1x, ${w2x} 2x`
}

const StreamsTab = () => {
    const { data: liveData, isLoading: liveLoading } = useQuery({
        queryKey: queryKeys.streams.live,
        queryFn: streamsApi.getLive,
        refetchInterval: 60 * 1000, // 1 minute
        staleTime: 1000 * 45, // 45 seconds
        placeholderData: keepPreviousData,
    })

    const { data: upcomingData, isLoading: upcomingLoading } = useQuery({
        queryKey: queryKeys.streams.upcoming,
        queryFn: streamsApi.getUpcoming,
        refetchInterval: 60 * 1000 * 5, // 5 minutes
        staleTime: 1000 * 60 * 4, // 4 minutes
        placeholderData: keepPreviousData,
    })

    return (
        <div className="space-y-6">
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <div className="flex items-center gap-2 mb-4">
                    <PlayCircle className="text-rose-500" />
                    <h3 className="text-lg font-bold text-slate-800">Live Streams</h3>
                    <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-rose-100 text-rose-600">
                        {liveData?.streams.length ?? 0}
                    </span>
                </div>

                {liveLoading ? (
                    <div className="h-40 flex items-center justify-center text-slate-400 text-sm">Loading...</div>
                ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {liveData?.streams.map((stream: Stream, index: number) => (
                            <a
                                key={stream.id}
                                href={stream.link || `https://www.youtube.com/watch?v=${stream.id}`}
                                target="_blank"
                                rel="noreferrer"
                                className="group relative block rounded-xl overflow-hidden border border-slate-200 hover:shadow-md transition-all"
                            >
                                {stream.thumbnail ? (
                                    <div className="aspect-video relative overflow-hidden bg-slate-100">
                                        <img
                                            src={getOptimizedThumbnail(stream.thumbnail)}
                                            srcSet={getThumbnailSrcSet(stream.thumbnail)}
                                            alt={stream.title}
                                            loading={index === 0 ? "eager" : "lazy"}
                                            decoding="async"
                                            fetchPriority={index === 0 ? "high" : "auto"}
                                            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                                            onError={(e) => {
                                                // \ucd5c\uc801\ud654 \uc2e4\ud328 \uc2dc \uc6d0\ubcf8 URL\ub85c fallback
                                                if (stream.thumbnail && e.currentTarget.src !== stream.thumbnail) {
                                                    e.currentTarget.src = stream.thumbnail;
                                                    e.currentTarget.srcset = '';
                                                } else {
                                                    e.currentTarget.style.display = 'none';
                                                }
                                            }}
                                        />
                                        <div className="absolute top-2 right-2 bg-rose-600 text-white text-[10px] font-bold px-1.5 py-0.5 rounded flex items-center gap-1 shadow-sm">
                                            LIVE
                                        </div>
                                    </div>
                                ) : (
                                    <div className="aspect-video bg-slate-100 flex items-center justify-center text-slate-300">
                                        <PlayCircle size={32} />
                                    </div>
                                )}
                                <div className="p-4">
                                    <h4 className="font-bold text-sm line-clamp-2 mb-1 text-slate-800">{stream.title}</h4>
                                    <p className="text-xs text-slate-500 mb-3">{stream.channel_name}</p>
                                    <span className="inline-flex items-center text-xs font-medium text-red-600 group-hover:text-red-700 group-hover:underline">
                                        <ExternalLink size={12} className="mr-1" /> Watch on YouTube
                                    </span>
                                </div>
                            </a>
                        ))}
                        {(!liveData?.streams || liveData.streams.length === 0) && (
                            <p className="col-span-full text-center text-slate-400 text-sm py-10">No live streams currently.</p>
                        )}
                    </div>
                )}
            </div>

            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <div className="flex items-center gap-2 mb-4">
                    <Calendar className="text-sky-500" />
                    <h3 className="text-lg font-bold text-slate-800">Upcoming Streams (24h)</h3>
                    <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-sky-100 text-sky-600">
                        {upcomingData?.streams.length ?? 0}
                    </span>
                </div>

                {upcomingLoading ? (
                    <div className="h-40 flex items-center justify-center text-slate-400 text-sm">Loading...</div>
                ) : (
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                        {upcomingData?.streams.map((stream: Stream) => (
                            <a
                                key={stream.id}
                                href={stream.link || `https://www.youtube.com/watch?v=${stream.id}`}
                                target="_blank"
                                rel="noreferrer"
                                className="flex items-center p-3 rounded-lg border border-slate-100 hover:bg-slate-50 transition-colors group"
                            >
                                <div className="w-20 h-14 rounded-lg overflow-hidden shrink-0 bg-slate-100 mr-4 relative flex items-center justify-center text-slate-300">
                                    {stream.thumbnail ? (
                                        <img
                                            src={getOptimizedThumbnail(stream.thumbnail, 160)}
                                            srcSet={`${getOptimizedThumbnail(stream.thumbnail, 160)} 1x, ${getOptimizedThumbnail(stream.thumbnail, 320)} 2x`}
                                            alt={stream.title}
                                            loading="lazy"
                                            decoding="async"
                                            className="w-full h-full object-cover"
                                            onError={(e) => {
                                                if (stream.thumbnail && e.currentTarget.src !== stream.thumbnail) {
                                                    e.currentTarget.src = stream.thumbnail;
                                                    e.currentTarget.srcset = '';
                                                } else {
                                                    e.currentTarget.style.display = 'none';
                                                }
                                            }}
                                        />
                                    ) : (
                                        <PlayCircle size={20} />
                                    )}
                                </div>
                                <div className="flex-1 min-w-0">
                                    <h4 className="font-medium text-sm text-slate-900 truncate group-hover:text-sky-600 transition-colors">{stream.title}</h4>
                                    <p className="text-xs text-slate-500 mt-0.5">{stream.channel_name}</p>
                                </div>
                                <div className="ml-4 text-right shrink-0 flex flex-col items-end gap-1">
                                    <div className="text-xs font-bold text-slate-700 bg-slate-100 px-2 py-1 rounded whitespace-nowrap">
                                        {stream.start_scheduled ? new Date(stream.start_scheduled).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : 'TBA'}
                                    </div>
                                    <span
                                        className="inline-flex items-center gap-1 text-[10px] text-red-600 hover:text-red-700 hover:bg-red-50 px-2 py-0.5 rounded transition-colors"
                                    >
                                        YouTube
                                        <ExternalLink size={10} />
                                    </span>
                                </div>
                            </a>
                        ))}
                        {(!upcomingData?.streams || upcomingData.streams.length === 0) && (
                            <p className="col-span-full text-center text-slate-400 text-sm py-10">No upcoming streams found.</p>
                        )}
                    </div>
                )}
            </div>
        </div>
    )
}

export default StreamsTab
