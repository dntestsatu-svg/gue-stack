const idrCurrencyFormatters: Record<number, Intl.NumberFormat> = {
  0: new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }),
  2: new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 2,
  }),
}

const shortDateTimeFormatter = new Intl.DateTimeFormat('id-ID', {
  dateStyle: 'short',
  timeStyle: 'short',
})

const mediumDateTimeFormatter = new Intl.DateTimeFormat('id-ID', {
  dateStyle: 'medium',
  timeStyle: 'short',
})

const mediumTimeFormatter = new Intl.DateTimeFormat('id-ID', {
  timeStyle: 'medium',
})

export function useFormatters() {
  const formatCurrency = (value: number, maximumFractionDigits = 0) => {
    const formatter =
      idrCurrencyFormatters[maximumFractionDigits] ??
      new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        maximumFractionDigits,
      })
    return formatter.format(value)
  }

  const formatPercent = (value: number, fractionDigits = 1) => `${value.toFixed(fractionDigits)}%`

  const formatDateShort = (value: string) => shortDateTimeFormatter.format(new Date(value))
  const formatDateMedium = (value: string) => mediumDateTimeFormatter.format(new Date(value))
  const formatTime = (value: string) => mediumTimeFormatter.format(new Date(value))

  return {
    formatCurrency,
    formatPercent,
    formatDateShort,
    formatDateMedium,
    formatTime,
  }
}
