import type { ComponentProps, ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface TabButtonProps extends ComponentProps<'button'> {
  active?: boolean
  icon?: string
  children: ReactNode
}

export function TabButton({
  active = false,
  icon,
  children,
  className,
  ...props
}: TabButtonProps) {
  return (
    <button
      className={cn(
        'py-4 px-1 border-b-2 font-medium text-sm transition-colors',
        active
          ? 'border-blue-500 text-blue-600'
          : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
        className
      )}
      {...props}
    >
      {icon && <span className="mr-2">{icon}</span>}
      {children}
    </button>
  )
}
