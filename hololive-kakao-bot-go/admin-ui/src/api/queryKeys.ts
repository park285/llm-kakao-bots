/**
 * Query Key Factory
 * TanStack Query의 queryKey를 중앙 관리하여 일관성 및 타입 안전성 확보
 *
 * @see https://tkdodo.eu/blog/effective-react-query-keys#use-query-key-factories
 */

export const queryKeys = {
    /** 멤버 관련 쿼리 키 */
    members: {
        all: ['members'] as const,
        detail: (id: number) => ['members', id] as const,
    },

    /** 알람 관련 쿼리 키 */
    alarms: {
        all: ['alarms'] as const,
    },

    /** 채팅방 관련 쿼리 키 */
    rooms: {
        all: ['rooms'] as const,
    },

    /** 통계 관련 쿼리 키 */
    stats: {
        summary: ['stats'] as const,
        channels: ['stats', 'channels'] as const,
    },

    /** 스트림 관련 쿼리 키 */
    streams: {
        live: ['streams', 'live'] as const,
        upcoming: ['streams', 'upcoming'] as const,
    },

    /** 로그 관련 쿼리 키 */
    logs: {
        all: ['logs'] as const,
    },

    /** 설정 관련 쿼리 키 */
    settings: {
        all: ['settings'] as const,
    },

    /** Docker 관련 쿼리 키 */
    docker: {
        health: ['docker-health'] as const,
        containers: ['docker-containers'] as const,
    },

    /** 마일스톤 관련 쿼리 키 */
    milestones: {
        all: ['milestones'] as const,
        near: ['milestones', 'near'] as const,
        stats: ['milestones', 'stats'] as const,
    },
} as const

/** 타입 추출용 */
export type QueryKeys = typeof queryKeys
