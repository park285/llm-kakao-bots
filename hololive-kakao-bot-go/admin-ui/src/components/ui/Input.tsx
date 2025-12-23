import { forwardRef, type ComponentProps } from 'react'
import { cn } from '@/lib/utils'

interface InputProps extends ComponentProps<'input'> {
  hasError?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ hasError = false, className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        'px-2 py-1 text-sm border rounded transition-colors',
        hasError
          ? 'border-red-500 focus:ring-red-500'
          : 'border-gray-300 focus:ring-blue-500',
        'focus:outline-none focus:ring-2',
        className
      )}
      {...props}
    />
  )
)

Input.displayName = 'Input'
