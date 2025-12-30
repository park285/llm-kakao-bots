/**
 * BaseModal - 공통 모달 컴포넌트
 * Headless UI의 Dialog + Transition 패턴을 중앙화
 */

import { Fragment } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import clsx from 'clsx'

interface BaseModalProps {
    isOpen: boolean
    onClose: () => void
    title?: React.ReactNode
    children: React.ReactNode
    /** 모달 최대 너비 (기본: max-w-md) */
    maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl'
    /** 타이틀 영역 커스텀 클래스 */
    titleClassName?: string
    /** 헤더 영역에 테두리 표시 여부 */
    showHeaderBorder?: boolean
}

const maxWidthClasses = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-2xl',
}

export const BaseModal = ({
    isOpen,
    onClose,
    title,
    children,
    maxWidth = 'md',
    titleClassName,
    showHeaderBorder = false,
}: BaseModalProps) => {
    return (
        <Transition appear show={isOpen} as={Fragment}>
            <Dialog as="div" className="relative z-50" onClose={onClose}>
                {/* Backdrop */}
                <Transition.Child
                    as={Fragment}
                    enter="ease-out duration-300"
                    enterFrom="opacity-0"
                    enterTo="opacity-100"
                    leave="ease-in duration-200"
                    leaveFrom="opacity-100"
                    leaveTo="opacity-0"
                >
                    <div className="fixed inset-0 bg-black/25 backdrop-blur-sm" />
                </Transition.Child>

                {/* Modal Container */}
                <div className="fixed inset-0 overflow-y-auto">
                    <div className="flex min-h-full items-center justify-center p-4 text-center">
                        <Transition.Child
                            as={Fragment}
                            enter="ease-out duration-300"
                            enterFrom="opacity-0 scale-95"
                            enterTo="opacity-100 scale-100"
                            leave="ease-in duration-200"
                            leaveFrom="opacity-100 scale-100"
                            leaveTo="opacity-0 scale-95"
                        >
                            <Dialog.Panel
                                className={clsx(
                                    'w-full transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all border border-slate-100',
                                    maxWidthClasses[maxWidth]
                                )}
                            >
                                {title && (
                                    <Dialog.Title
                                        as="h3"
                                        className={clsx(
                                            'text-lg font-bold leading-6 text-slate-900',
                                            showHeaderBorder && 'border-b border-slate-100 pb-4 mb-4',
                                            titleClassName
                                        )}
                                    >
                                        {title}
                                    </Dialog.Title>
                                )}
                                {children}
                            </Dialog.Panel>
                        </Transition.Child>
                    </div>
                </div>
            </Dialog>
        </Transition>
    )
}
