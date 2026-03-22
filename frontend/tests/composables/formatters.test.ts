import { describe, expect, it } from 'vitest'
import { useFormatters } from '@/composables/useFormatters'

describe('useFormatters', () => {
  it('formats currency and percent consistently', () => {
    const { formatCurrency, formatPercent } = useFormatters()

    expect(formatCurrency(1000)).toContain('Rp')
    expect(formatCurrency(1000, 2)).toContain('Rp')
    expect(formatPercent(12.345, 2)).toBe('12.35%')
  })

  it('formats date and time values', () => {
    const { formatDateShort, formatDateMedium, formatTime } = useFormatters()
    const sample = '2026-03-21T10:00:00Z'

    expect(formatDateShort(sample).length).toBeGreaterThan(0)
    expect(formatDateMedium(sample).length).toBeGreaterThan(0)
    expect(formatTime(sample).length).toBeGreaterThan(0)
  })
})
