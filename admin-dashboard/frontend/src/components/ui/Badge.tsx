import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"
import { X } from 'lucide-react'

const badgeVariants = cva(
  "inline-flex items-center rounded-md px-2.5 py-1 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "border-transparent bg-slate-100 text-slate-900 hover:bg-slate-200",
        destructive:
          "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
        outline: "text-foreground border border-slate-200",
        // 레거시 color prop 호환용 스타일
        blue: 'border-transparent bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-700/10',
        sky: 'border-transparent bg-sky-50 text-sky-700 ring-1 ring-inset ring-sky-700/10',
        indigo: 'border-transparent bg-indigo-50 text-indigo-700 ring-1 ring-inset ring-indigo-700/10',
        green: 'border-transparent bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-700/10',
        yellow: 'border-transparent bg-yellow-50 text-yellow-800 ring-1 ring-inset ring-yellow-600/20',
        amber: 'border-transparent bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20',
        red: 'border-transparent bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/10',
        rose: 'border-transparent bg-rose-50 text-rose-700 ring-1 ring-inset ring-rose-600/10',
        gray: 'border-transparent bg-slate-50 text-slate-600 ring-1 ring-inset ring-slate-500/10',
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

// badgeVariants에서 허용된 variant 값 추출
type BadgeVariant = NonNullable<VariantProps<typeof badgeVariants>['variant']>

// 허용된 variant 값 목록 (타입 가드용)
const validBadgeVariants = [
  'default', 'secondary', 'destructive', 'outline',
  'blue', 'sky', 'indigo', 'green', 'yellow', 'amber', 'red', 'rose', 'gray'
] as const

// color prop이 유효한 variant인지 검증하는 타입 가드
const isValidBadgeVariant = (value: string): value is BadgeVariant =>
  (validBadgeVariants as readonly string[]).includes(value)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
  VariantProps<typeof badgeVariants> {
  /** 레거시 호환성을 위한 color prop (variant 우선) */
  color?: BadgeVariant
  onRemove?: () => void
}

function Badge({ className, variant, color, onRemove, children, ...props }: BadgeProps) {
  // 타입 안전한 variant 결정 로직
  // 1. variant prop이 있으면 우선 사용
  // 2. color prop이 유효한 variant면 사용
  // 3. 그 외에는 기본값 'default'
  let finalVariant: BadgeVariant = 'default'
  if (variant) {
    finalVariant = variant
  } else if (color && isValidBadgeVariant(color)) {
    finalVariant = color
  }

  return (
    <div className={cn(badgeVariants({ variant: finalVariant }), className)} {...props}>
      {children}
      {onRemove && (
        <button
          onClick={onRemove}
          className="ml-1 group rounded-full p-0.5 hover:bg-black/5 transition-colors"
          type="button"
          aria-label="Remove"
        >
          <X size={12} strokeWidth={3} className="opacity-60 group-hover:opacity-100" />
        </button>
      )}
    </div>
  )
}

export { Badge, badgeVariants }
