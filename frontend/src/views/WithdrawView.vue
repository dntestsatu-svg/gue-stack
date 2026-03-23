<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ArrowUpRight, Landmark, RefreshCcw, Store, TriangleAlert } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useFormatters } from '@/composables/useFormatters'
import { getApiErrorMessage } from '@/services/http'
import * as withdrawApi from '@/services/withdraw'
import type {
  WithdrawBankOption,
  WithdrawHistoryPage,
  WithdrawInquiryResult,
  WithdrawOptionsResult,
  WithdrawTokoOption,
  WithdrawTransferResult,
} from '@/services/types'

const { formatCurrency, formatDateMedium } = useFormatters()

const options = ref<WithdrawOptionsResult | null>(null)
const historyPage = ref<WithdrawHistoryPage | null>(null)
const loading = ref(false)
const historyLoading = ref(false)
const submitting = ref(false)
const pageErrorMessage = ref('')
const lastUpdated = ref('')

const selectedTokoID = ref('')
const selectedBankID = ref('')

const withdrawForm = reactive({
  amount: '25000',
})

const inquiryResult = ref<WithdrawInquiryResult | null>(null)
const transferResult = ref<WithdrawTransferResult | null>(null)
const historyPagination = reactive({
  limit: 10,
  offset: 0,
  total: 0,
  hasMore: false,
})

const tokos = computed(() => options.value?.tokos ?? [])
const banks = computed(() => options.value?.banks ?? [])
const historyItems = computed(() => historyPage.value?.items ?? [])
const selectedToko = computed<WithdrawTokoOption | null>(() =>
  tokos.value.find((item) => String(item.id) === selectedTokoID.value) ?? null,
)
const selectedBank = computed<WithdrawBankOption | null>(() =>
  banks.value.find((item) => String(item.id) === selectedBankID.value) ?? null,
)
const canSubmit = computed(() =>
  Boolean(selectedToko.value && selectedBank.value && Number(withdrawForm.amount) >= 25000),
)

const historyRangeLabel = computed(() => {
  if (historyPagination.total === 0 || historyItems.value.length === 0) {
    return 'No withdraw requests yet'
  }
  const start = historyPagination.offset + 1
  const end = Math.min(historyPagination.offset + historyItems.value.length, historyPagination.total)
  return `Showing ${start}-${end} of ${historyPagination.total} withdraw requests`
})

const historyCurrentPage = computed(() => Math.floor(historyPagination.offset / historyPagination.limit) + 1)

const loadOptions = async () => {
  const result = await withdrawApi.fetchOptions()
  options.value = result
  if (!selectedTokoID.value && result.tokos.length > 0) {
    selectedTokoID.value = String(result.tokos[0].id)
  }
  if (!selectedBankID.value && result.banks.length > 0) {
    selectedBankID.value = String(result.banks[0].id)
  }
}

const loadHistory = async () => {
  historyLoading.value = true
  try {
    const result = await withdrawApi.fetchHistory({
      limit: historyPagination.limit,
      offset: historyPagination.offset,
    })
    historyPage.value = result
    historyPagination.total = result.total
    historyPagination.limit = result.limit
    historyPagination.offset = result.offset
    historyPagination.hasMore = result.has_more
  } finally {
    historyLoading.value = false
  }
}

const loadPageData = async () => {
  loading.value = true
  pageErrorMessage.value = ''
  try {
    await Promise.all([loadOptions(), loadHistory()])
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    pageErrorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const resetResults = () => {
  inquiryResult.value = null
  transferResult.value = null
}

const submitWithdraw = async () => {
  if (!selectedToko.value || !selectedBank.value) {
    toast.error('Pilih toko dan bank tujuan terlebih dahulu')
    return
  }

  const amount = Number(withdrawForm.amount)
  if (!Number.isFinite(amount) || amount < 25000) {
    toast.error('Minimal withdraw Rp25.000')
    return
  }

  submitting.value = true
  pageErrorMessage.value = ''
  resetResults()
  try {
    const inquiry = await withdrawApi.inquiry({
      toko_id: selectedToko.value.id,
      bank_id: selectedBank.value.id,
      amount,
    })
    inquiryResult.value = inquiry
    toast.success(`Rekening terverifikasi atas nama ${inquiry.account_name}`)

    const transfer = await withdrawApi.transfer({
      toko_id: selectedToko.value.id,
      bank_id: selectedBank.value.id,
      amount,
      inquiry_id: inquiry.inquiry_id,
    })
    transferResult.value = transfer
    toast.success(transfer.message)
    historyPagination.offset = 0
    await loadPageData()
  } catch (error) {
    const message = getApiErrorMessage(error)
    pageErrorMessage.value = message
    toast.error(message || 'error')
  } finally {
    submitting.value = false
  }
}

const nextHistoryPage = async () => {
  if (!historyPagination.hasMore || historyLoading.value) {
    return
  }
  const previousOffset = historyPagination.offset
  historyPagination.offset += historyPagination.limit
  try {
    await loadHistory()
  } catch (error) {
    historyPagination.offset = previousOffset
    pageErrorMessage.value = getApiErrorMessage(error)
  }
}

const prevHistoryPage = async () => {
  if (historyPagination.offset <= 0 || historyLoading.value) {
    return
  }
  const previousOffset = historyPagination.offset
  historyPagination.offset = Math.max(historyPagination.offset - historyPagination.limit, 0)
  try {
    await loadHistory()
  } catch (error) {
    historyPagination.offset = previousOffset
    pageErrorMessage.value = getApiErrorMessage(error)
  }
}

const withdrawStatusVariant = (status: string) => {
  const normalized = status.trim().toLowerCase()
  if (normalized === 'success') {
    return 'default'
  }
  if (normalized === 'pending') {
    return 'secondary'
  }
  if (normalized === 'failed') {
    return 'destructive'
  }
  return 'outline'
}

const formatHistoryDate = (value: string) => formatDateMedium(value)

void loadPageData()
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Settlement Transfer"
      title="Withdraw"
      description="Tarik saldo settlement toko yang ada dalam scope akun anda ke rekening bank yang tersimpan pada profil user."
      :updated-at="lastUpdated"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="loading || submitting" @click="loadPageData">
          <RefreshCcw class="mr-2 h-4 w-4" />
          {{ loading ? 'Refreshing...' : 'Refresh Data' }}
        </Button>
      </template>
    </PageHeader>

    <Alert v-if="pageErrorMessage" variant="destructive">
      <TriangleAlert class="h-4 w-4" />
      <AlertTitle>Withdraw Request Failed</AlertTitle>
      <AlertDescription>{{ pageErrorMessage }}</AlertDescription>
    </Alert>

    <div v-if="loading && !options" class="space-y-4">
      <Card class="app-panel">
        <CardContent class="space-y-3 p-6">
          <Skeleton class="h-6 w-56" />
          <Skeleton class="h-11 w-full" />
          <Skeleton class="h-11 w-full" />
        </CardContent>
      </Card>
    </div>

    <template v-else>
      <Card v-if="tokos.length === 0 || banks.length === 0" class="app-panel">
        <CardContent class="p-6">
          <EmptyState
            title="Withdraw Belum Siap"
            :description="tokos.length === 0
              ? 'Belum ada toko dalam scope akun anda yang bisa dipakai untuk withdraw.'
              : 'Belum ada rekening bank milik user ini. Tambahkan dulu bank di Bank Management.'"
          />
        </CardContent>
      </Card>

      <template v-else>
        <div class="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
          <Card class="app-panel">
            <CardHeader>
              <CardTitle>Withdraw Request</CardTitle>
              <CardDescription>
                Sistem akan inquiry dulu ke gateway untuk validasi rekening, lalu langsung meneruskan request transfer jika valid.
              </CardDescription>
            </CardHeader>
            <CardContent class="space-y-4">
              <div class="grid gap-4 md:grid-cols-2">
                <div class="min-w-0 space-y-2">
                  <Label for="withdraw-toko">Toko</Label>
                  <Select v-model="selectedTokoID">
                    <SelectTrigger id="withdraw-toko" class="w-full">
                      <SelectValue placeholder="Pilih toko" />
                    </SelectTrigger>
                    <SelectContent class="w-(--reka-select-trigger-width) max-w-(--reka-select-trigger-width)">
                      <SelectItem v-for="item in tokos" :key="item.id" :value="String(item.id)">
                        {{ item.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="min-w-0 space-y-2">
                  <Label for="withdraw-bank">Bank Tujuan</Label>
                  <Select v-model="selectedBankID">
                    <SelectTrigger id="withdraw-bank" class="w-full">
                      <SelectValue placeholder="Pilih bank" />
                    </SelectTrigger>
                    <SelectContent class="w-(--reka-select-trigger-width) max-w-(--reka-select-trigger-width)">
                      <SelectItem v-for="item in banks" :key="item.id" :value="String(item.id)">
                        {{ item.bank_name }} • {{ item.account_name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2 md:col-span-2">
                  <Label for="withdraw-amount">Nominal Withdraw</Label>
                  <Input id="withdraw-amount" v-model="withdrawForm.amount" type="number" min="25000" step="1000" />
                </div>
              </div>

              <div class="grid gap-3 md:grid-cols-2">
                <div class="rounded-xl border bg-(--background-muted) p-4">
                  <div class="flex items-start gap-3">
                    <Store class="mt-0.5 h-4 w-4 text-primary" />
                    <div class="space-y-1">
                      <p class="text-sm font-medium text-foreground">{{ selectedToko?.name || '-' }}</p>
                      <p class="text-xs text-muted-foreground">Settle Balance</p>
                      <p class="text-lg font-semibold text-foreground">
                        {{ formatCurrency(selectedToko?.settle_balance ?? 0) }}
                      </p>
                    </div>
                  </div>
                </div>

                <div class="rounded-xl border bg-(--background-muted) p-4">
                  <div class="flex items-start gap-3">
                    <Landmark class="mt-0.5 h-4 w-4 text-primary" />
                    <div class="space-y-1">
                      <p class="text-sm font-medium text-foreground">{{ selectedBank?.bank_name || '-' }}</p>
                      <p class="text-xs text-muted-foreground">{{ selectedBank?.account_name || '-' }}</p>
                      <p class="text-sm text-foreground">{{ selectedBank?.account_number || '-' }}</p>
                    </div>
                  </div>
                </div>
              </div>

              <div class="flex justify-end">
                <Button :disabled="submitting || !canSubmit" @click="submitWithdraw">
                  <Spinner v-if="submitting" class="mr-2" />
                  <ArrowUpRight v-else class="mr-2 h-4 w-4" />
                  {{ submitting ? 'Processing Withdraw...' : 'Request Withdraw' }}
                </Button>
              </div>
            </CardContent>
          </Card>

          <div class="space-y-4">
            <Card class="dashboard-kpi-card">
              <CardHeader class="pb-2">
                <CardDescription>Selected Settle Balance</CardDescription>
                <CardTitle class="text-2xl">{{ formatCurrency(selectedToko?.settle_balance ?? 0) }}</CardTitle>
              </CardHeader>
            </Card>

            <Card v-if="inquiryResult" class="app-panel">
              <CardHeader>
                <CardTitle>Inquiry Verified</CardTitle>
                <CardDescription>Nama rekening berhasil diverifikasi sebelum transfer dikirim.</CardDescription>
              </CardHeader>
              <CardContent class="space-y-3">
                <div class="flex flex-wrap gap-2">
                  <Badge variant="outline">{{ inquiryResult.bank_name }}</Badge>
                  <Badge variant="secondary">Inquiry ID {{ inquiryResult.inquiry_id }}</Badge>
                </div>
                <div class="rounded-xl border bg-(--background-muted) p-4 text-sm">
                  <p><strong>{{ inquiryResult.account_name }}</strong></p>
                  <p class="text-muted-foreground">{{ inquiryResult.account_number }}</p>
                  <p class="mt-2 text-muted-foreground">Fee gateway: {{ formatCurrency(inquiryResult.fee) }}</p>
                </div>
              </CardContent>
            </Card>

            <Alert v-if="transferResult" variant="default">
              <ArrowUpRight class="h-4 w-4" />
              <AlertTitle>{{ transferResult.message }}</AlertTitle>
              <AlertDescription>
                Withdraw {{ formatCurrency(transferResult.amount) }} ke {{ transferResult.bank_name }}
                atas nama {{ transferResult.account_name }} sudah diteruskan.
              </AlertDescription>
            </Alert>
          </div>
        </div>

        <Card class="app-panel">
          <CardHeader>
            <CardTitle>Withdraw Request History</CardTitle>
            <CardDescription>
              Status terbaru request withdraw scoped ke toko yang memang bisa diakses akun anda.
            </CardDescription>
          </CardHeader>
          <CardContent class="app-table-shell">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Requested At</TableHead>
                  <TableHead>Toko</TableHead>
                  <TableHead>Reference</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead class="text-right">Amount</TableHead>
                  <TableHead class="text-right">Netto</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <template v-if="historyItems.length > 0">
                  <TableRow v-for="item in historyItems" :key="item.id">
                    <TableCell>{{ formatHistoryDate(item.created_at) }}</TableCell>
                    <TableCell class="font-medium">{{ item.toko_name }}</TableCell>
                    <TableCell class="font-mono text-xs text-muted-foreground">
                      {{ item.reference || '-' }}
                    </TableCell>
                    <TableCell>
                      <Badge :variant="withdrawStatusVariant(item.status)" class="capitalize">
                        {{ item.status }}
                      </Badge>
                    </TableCell>
                    <TableCell class="text-right">{{ formatCurrency(item.amount) }}</TableCell>
                    <TableCell class="text-right">{{ formatCurrency(item.netto) }}</TableCell>
                  </TableRow>
                </template>
                <TableEmpty v-else :colspan="6">
                  <EmptyState
                    title="Belum Ada Histori Withdraw"
                    description="Request withdraw yang sudah dikirim akan tampil di sini beserta status pending, success, atau failed."
                  />
                </TableEmpty>
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border bg-(--background-elevated) px-4 py-3">
          <div class="space-y-1">
            <p class="text-sm font-medium text-foreground">{{ historyRangeLabel }}</p>
            <p class="text-xs text-muted-foreground">Page {{ historyCurrentPage }} • Limit {{ historyPagination.limit }}</p>
          </div>
          <div class="flex items-center gap-2">
            <Button size="sm" variant="outline" :disabled="historyLoading || historyPagination.offset <= 0" @click="prevHistoryPage">
              Prev
            </Button>
            <Button size="sm" variant="outline" :disabled="historyLoading || !historyPagination.hasMore" @click="nextHistoryPage">
              Next
            </Button>
          </div>
        </div>
      </template>
    </template>
  </section>
</template>
