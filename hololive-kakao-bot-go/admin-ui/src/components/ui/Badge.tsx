import type { ComponentProps, ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { X } from 'lucide-react'

type BadgeVariant = 'blue' | 'green' | 'yellow' | 'red' | 'gray' | 'sky' | 'indigo' | 'rose' | 'amber'

interface BadgeProps extends ComponentProps<'span'> {
  color?: BadgeVariant
  children: ReactNode
  onRemove?: () => void
}

const BADGE_VARIANTS: Record<BadgeVariant, string> = {
  blue: 'bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-700/10',
  sky: 'bg-sky-50 text-sky-700 ring-1 ring-inset ring-sky-700/10',
  indigo: 'bg-indigo-50 text-indigo-700 ring-1 ring-inset ring-indigo-700/10',
  green: 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-700/10',
  yellow: 'bg-yellow-50 text-yellow-800 ring-1 ring-inset ring-yellow-600/20',
  amber: 'bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20',
  red: 'bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/10',
  rose: 'bg-rose-50 text-rose-700 ring-1 ring-inset ring-rose-600/10',
  gray: 'bg-slate-50 text-slate-600 ring-1 ring-inset ring-slate-500/10',
}

export function Badge({
  color = 'blue',
  children,
  onRemove,
  className,
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold rounded-md transition-colors',
        BADGE_VARIANTS[color],
        className
      )}
      {...props}
    >
      {children}
      {onRemove && (
        <button
          onClick={onRemove}
          className="group rounded-full p-0.5 hover:bg-black/5 transition-colors"
          type="button"
          aria-label="Remove"
        >
          <X size={12} strokeWidth={3} className="opacity-60 group-hover:opacity-100" />
        </button>
      )}
    </span>
  )
}
