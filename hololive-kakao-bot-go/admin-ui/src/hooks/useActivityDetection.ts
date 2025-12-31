import { useState, useEffect, useRef, useCallback } from 'react'

const CHANNEL_NAME = 'admin_session'

interface TabSyncMessage {
    type: 'ACTIVITY' | 'LOGOUT'
    timestamp: number
}

/**
 * 사용자 활동 감지 훅 (멀티 탭 동기화 포함)
 *
 * - 현재 탭에서 활동 감지 시 다른 탭에도 BroadcastChannel로 알림
 * - 다른 탭에서 활동 알림 수신 시 현재 탭의 Idle 타이머도 리셋
 * - 이를 통해 "모든 탭이 동시에 Idle 상태일 때만" idle=true 전송 (팀킬 방지)
 *
 * @param idleTimeoutMs 유휴 상태로 간주할 시간 (밀리초)
 * @returns isIdle: 유휴 상태 여부
 */
export function useActivityDetection(idleTimeoutMs: number) {
    const [isIdle, setIsIdle] = useState(false)
    const timeoutRef = useRef<number | null>(null)
    const channelRef = useRef<BroadcastChannel | null>(null)

    // 타이머 리셋 (로컬 전용, 브로드캐스트 안 함)
    const resetTimerInternal = useCallback(() => {
        setIsIdle(false)

        if (timeoutRef.current) {
            window.clearTimeout(timeoutRef.current)
        }

        timeoutRef.current = window.setTimeout(() => {
            setIsIdle(true)
        }, idleTimeoutMs)
    }, [idleTimeoutMs])

    // 타이머 리셋 + 다른 탭에 브로드캐스트
    const resetTimer = useCallback(() => {
        resetTimerInternal()

        // 다른 탭에 활동 알림 (BroadcastChannel)
        if (channelRef.current) {
            const message: TabSyncMessage = {
                type: 'ACTIVITY',
                timestamp: Date.now(),
            }
            channelRef.current.postMessage(message)
        }
    }, [resetTimerInternal])

    // BroadcastChannel 설정 (다른 탭에서 활동 수신 시 타이머 리셋)
    useEffect(() => {
        if (typeof BroadcastChannel === 'undefined') {
            // BroadcastChannel 미지원 브라우저
            return
        }

        channelRef.current = new BroadcastChannel(CHANNEL_NAME)

        channelRef.current.onmessage = (event: MessageEvent<TabSyncMessage>) => {
            if (event.data.type === 'ACTIVITY') {
                // 다른 탭에서 활동 감지 → 현재 탭 타이머 리셋 (브로드캐스트 안 함)
                resetTimerInternal()
            }
        }

        return () => {
            channelRef.current?.close()
            channelRef.current = null
        }
    }, [resetTimerInternal])

    // 이벤트 리스너 설정
    useEffect(() => {
        const events = ['mousemove', 'keydown', 'click', 'scroll', 'touchstart']

        events.forEach(event => document.addEventListener(event, resetTimer, { passive: true }))

        // 초기 타이머 시작
        resetTimerInternal()

        return () => {
            events.forEach(event => document.removeEventListener(event, resetTimer))
            if (timeoutRef.current) {
                window.clearTimeout(timeoutRef.current)
            }
        }
    }, [resetTimer, resetTimerInternal])

    return isIdle
}
