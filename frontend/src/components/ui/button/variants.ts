import { cva, type VariantProps } from 'class-variance-authority'

export const buttonVariants = cva(
  'inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand)] disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-[var(--brand)] text-white hover:bg-[var(--brand-strong)]',
        outline: 'border border-[var(--border)] bg-[var(--background-elevated)] text-[var(--foreground)] hover:bg-[var(--background-muted)]',
        ghost: 'text-[var(--muted-foreground)] hover:bg-[var(--background-muted)] hover:text-[var(--foreground)]',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 px-3',
        lg: 'h-11 px-6',
        icon: 'h-9 w-9',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export type ButtonVariants = VariantProps<typeof buttonVariants>
