import { useState, useEffect, useRef, useCallback } from 'react'

/**
 * 사용자 활동 감지 훅
 * @param idleTimeoutMs 유휴 상태로 간주할 시간 (밀리초)
 * @returns isIdle: 유휴 상태 여부
 */
export function useActivityDetection(idleTimeoutMs: number) {
    const [isIdle, setIsIdle] = useState(false)
    const timeoutRef = useRef<number | null>(null)

    const resetTimer = useCallback(() => {
        // 이미 idle 상태였다면 false로 변경 (활동 감지)
        setIsIdle(false)

        if (timeoutRef.current) {
            window.clearTimeout(timeoutRef.current)
        }

        // 타임아웃 재설정
        timeoutRef.current = window.setTimeout(() => {
            setIsIdle(true)
        }, idleTimeoutMs)
    }, [idleTimeoutMs])

    useEffect(() => {
        // 감지할 이벤트 목록
        const events = ['mousemove', 'keydown', 'click', 'scroll', 'touchstart']

        // 이벤트 리스너 등록 (passive: true로 성능 최적화)
        events.forEach(event => document.addEventListener(event, resetTimer, { passive: true }))

        // 초기 타이머 시작
        resetTimer()

        // Cleanup
        return () => {
            events.forEach(event => document.removeEventListener(event, resetTimer))
            if (timeoutRef.current) {
                window.clearTimeout(timeoutRef.current)
            }
        }
    }, [resetTimer])

    return isIdle
}
