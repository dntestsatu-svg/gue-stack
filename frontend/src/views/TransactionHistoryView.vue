<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Transaction Intelligence"
      title="Histori Transaksi"
      description="Server-side filtering, search, pagination, dan export untuk dataset besar."
      :updated-at="lastUpdated"
    >
      <template #actions>
        <Button size="sm" variant="outline" :disabled="loading" @click="loadHistory">
          {{ loading ? 'Loading...' : 'Refresh' }}
        </Button>
        <Button size="sm" variant="outline" :disabled="exportingCSV || loading" @click="downloadExport('csv')">
          {{ exportingCSV ? 'Exporting...' : 'Export CSV' }}
        </Button>
        <Button size="sm" :disabled="exportingDOCX || loading" @click="downloadExport('docx')">
          {{ exportingDOCX ? 'Exporting...' : 'Export DOCX' }}
        </Button>
      </template>
    </PageHeader>

    <Card class="app-panel">
      <CardHeader>
        <CardTitle>Filters</CardTitle>
        <CardDescription>
          Search berdasarkan reference, player, atau code. Rentang tanggal menggunakan UTC.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid gap-3 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.25fr)]">
          <div class="space-y-2">
            <Label for="trx-search">Search</Label>
            <Input
              id="trx-search"
              v-model="filters.q"
              placeholder="reference / player / code"
              @keydown.enter.prevent="applyFilters"
            />
          </div>
          <div class="space-y-2">
            <DateRangePicker v-model="dateRange" :disabled="loading" />
          </div>
        </div>
        <div class="flex flex-wrap items-center justify-end gap-2">
          <Button variant="ghost" size="sm" :disabled="loading" @click="resetFilters">Reset</Button>
          <Button size="sm" :disabled="loading" @click="applyFilters">Apply Filters</Button>
        </div>
      </CardContent>
    </Card>

    <Card class="app-panel">
      <CardHeader>
        <CardTitle>Latest Order (All Status)</CardTitle>
        <CardDescription>{{ rangeLabel }}</CardDescription>
      </CardHeader>
      <CardContent>
        <p v-if="errorMessage" class="mb-4 rounded-md border border-(--danger)/25 bg-(--danger)/8 px-3 py-2 text-sm text-(--danger)">
          {{ errorMessage }}
        </p>

        <div class="app-table-shell">
          <table class="app-data-table min-w-270 text-sm">
            <thead>
              <tr class="border-b border-border text-left text-muted-foreground">
                <th class="px-3 py-3 font-medium">Waktu</th>
                <th class="px-3 py-3 font-medium">Toko</th>
                <th class="px-3 py-3 font-medium">Player</th>
                <th class="px-3 py-3 font-medium">Code</th>
                <th class="px-3 py-3 font-medium">Reference</th>
                <th class="px-3 py-3 font-medium">Type</th>
                <th class="px-3 py-3 font-medium">Status</th>
                <th class="px-3 py-3 font-medium text-right">Amount</th>
                <th class="px-3 py-3 font-medium text-right">Netto</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in history"
                :key="item.id"
                class="border-b border-border/70 transition hover:bg-(--background-muted)/40"
              >
                <td class="px-3 py-3">{{ formatDate(item.created_at) }}</td>
                <td class="px-3 py-3">{{ item.toko_name }}</td>
                <td class="px-3 py-3">{{ item.player || '-' }}</td>
                <td class="px-3 py-3">{{ item.code || '-' }}</td>
                <td class="px-3 py-3">{{ item.reference || '-' }}</td>
                <td class="px-3 py-3 capitalize">{{ item.type }}</td>
                <td class="px-3 py-3">
                  <span
                    class="status-pill"
                    :class="{
                      'bg-[color-mix(in_oklab,var(--success)_20%,transparent)] text-(--success)': item.status === 'success',
                      'bg-[color-mix(in_oklab,var(--warning)_20%,transparent)] text-(--warning)': item.status === 'pending',
                      'bg-[color-mix(in_oklab,var(--danger)_20%,transparent)] text-(--danger)': item.status !== 'success' && item.status !== 'pending',
                    }"
                  >
                    {{ item.status }}
                  </span>
                </td>
                <td class="px-3 py-3 text-right">{{ formatCurrency(item.amount) }}</td>
                <td class="px-3 py-3 text-right">{{ formatCurrency(item.netto) }}</td>
              </tr>
              <tr v-if="!loading && history.length === 0">
                <td colspan="9" class="px-3 py-8 text-center text-muted-foreground">Belum ada transaksi untuk filter ini.</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="mt-4 flex flex-wrap items-center justify-between gap-2">
          <p class="text-sm text-muted-foreground">
            Page {{ currentPage }} • Limit {{ pagination.limit }}
          </p>
          <div class="flex items-center gap-2">
            <Button size="sm" variant="outline" :disabled="loading || pagination.offset <= 0" @click="prevPage">
              Prev
            </Button>
            <Button size="sm" variant="outline" :disabled="loading || !pagination.hasMore" @click="nextPage">
              Next
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import DateRangePicker from '@/components/DateRangePicker.vue'
import { useFormatters } from '@/composables/useFormatters'
import { getApiErrorMessage } from '@/services/http'
import * as dashboardApi from '@/services/dashboard'
import type { TransactionHistoryItem, TransactionHistoryQuery } from '@/services/types'

const loading = ref(false)
const exportingCSV = ref(false)
const exportingDOCX = ref(false)
const history = ref<TransactionHistoryItem[]>([])
const errorMessage = ref('')
const lastUpdated = ref('')

const filters = reactive({
  q: '',
  from: '',
  to: '',
})

const pagination = reactive({
  limit: 20,
  offset: 0,
  total: 0,
  hasMore: false,
})

const dateRange = computed({
  get: () => ({
    from: filters.from,
    to: filters.to,
  }),
  set: (value: { from: string; to: string }) => {
    filters.from = value.from
    filters.to = value.to
  },
})

const { formatCurrency, formatDateMedium } = useFormatters()

const historyQuery = (): TransactionHistoryQuery => {
  const query: TransactionHistoryQuery = {
    limit: pagination.limit,
    offset: pagination.offset,
  }
  if (filters.q.trim() !== '') {
    query.q = filters.q.trim()
  }
  if (filters.from.trim() !== '') {
    query.from = filters.from.trim()
  }
  if (filters.to.trim() !== '') {
    query.to = filters.to.trim()
  }
  return query
}

const loadHistory = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await dashboardApi.fetchHistory(historyQuery())
    history.value = page.items
    pagination.total = page.total
    pagination.limit = page.limit
    pagination.offset = page.offset
    pagination.hasMore = page.has_more
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const applyFilters = async () => {
  pagination.offset = 0
  await loadHistory()
}

const resetFilters = async () => {
  filters.q = ''
  filters.from = ''
  filters.to = ''
  pagination.offset = 0
  await loadHistory()
}

const nextPage = async () => {
  if (!pagination.hasMore) {
    return
  }
  pagination.offset += pagination.limit
  await loadHistory()
}

const prevPage = async () => {
  if (pagination.offset <= 0) {
    return
  }
  pagination.offset = Math.max(pagination.offset - pagination.limit, 0)
  await loadHistory()
}

const downloadBlob = (blob: globalThis.Blob, fileName: string) => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return
  }
  const url = window.URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = fileName
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  window.URL.revokeObjectURL(url)
}

const downloadExport = async (format: 'csv' | 'docx') => {
  if (format === 'csv') {
    exportingCSV.value = true
  } else {
    exportingDOCX.value = true
  }

  errorMessage.value = ''
  try {
    const { blob, fileName } = await dashboardApi.exportHistory(format, {
      q: filters.q.trim() || undefined,
      from: filters.from.trim() || undefined,
      to: filters.to.trim() || undefined,
      limit: 10000,
      offset: 0,
    })
    downloadBlob(blob, fileName)
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    if (format === 'csv') {
      exportingCSV.value = false
    } else {
      exportingDOCX.value = false
    }
  }
}

const rangeLabel = computed(() => {
  if (pagination.total === 0 || history.value.length === 0) {
    return 'No data'
  }
  const start = pagination.offset + 1
  const end = Math.min(pagination.offset + history.value.length, pagination.total)
  return `Showing ${start}-${end} of ${pagination.total}`
})

const currentPage = computed(() => Math.floor(pagination.offset / pagination.limit) + 1)
const formatDate = (value: string) => formatDateMedium(value)

void loadHistory()
</script>
