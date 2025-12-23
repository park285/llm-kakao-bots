import type { ComponentProps, ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface CardProps extends ComponentProps<'div'> {
  children: ReactNode
  hover?: boolean
}

interface CardHeaderProps extends ComponentProps<'div'> {
  children: ReactNode
}

interface CardBodyProps extends ComponentProps<'div'> {
  children: ReactNode
}

function CardRoot({ children, hover = false, className, ...props }: CardProps) {
  return (
    <div
      className={cn(
        'bg-white border border-gray-200 rounded-lg p-4',
        hover && 'hover:shadow-md transition-shadow',
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}

function CardHeader({ children, className, ...props }: CardHeaderProps) {
  return (
    <div className={cn('mb-3', className)} {...props}>
      {children}
    </div>
  )
}

function CardBody({ children, className, ...props }: CardBodyProps) {
  return (
    <div className={cn('space-y-3', className)} {...props}>
      {children}
    </div>
  )
}

// Compound Component Export
export const Card = Object.assign(CardRoot, {
  Header: CardHeader,
  Body: CardBody,
})
