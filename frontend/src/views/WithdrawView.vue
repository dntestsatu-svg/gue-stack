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
import { useFormatters } from '@/composables/useFormatters'
import { getApiErrorMessage } from '@/services/http'
import * as withdrawApi from '@/services/withdraw'
import type { WithdrawBankOption, WithdrawInquiryResult, WithdrawOptionsResult, WithdrawTokoOption, WithdrawTransferResult } from '@/services/types'

const { formatCurrency } = useFormatters()

const options = ref<WithdrawOptionsResult | null>(null)
const loading = ref(false)
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

const tokos = computed(() => options.value?.tokos ?? [])
const banks = computed(() => options.value?.banks ?? [])
const selectedToko = computed<WithdrawTokoOption | null>(() =>
  tokos.value.find((item) => String(item.id) === selectedTokoID.value) ?? null,
)
const selectedBank = computed<WithdrawBankOption | null>(() =>
  banks.value.find((item) => String(item.id) === selectedBankID.value) ?? null,
)
const canSubmit = computed(() =>
  Boolean(selectedToko.value && selectedBank.value && Number(withdrawForm.amount) >= 25000),
)

const loadOptions = async () => {
  loading.value = true
  pageErrorMessage.value = ''
  try {
    const result = await withdrawApi.fetchOptions()
    options.value = result
    if (!selectedTokoID.value && result.tokos.length > 0) {
      selectedTokoID.value = String(result.tokos[0].id)
    }
    if (!selectedBankID.value && result.banks.length > 0) {
      selectedBankID.value = String(result.banks[0].id)
    }
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
    await loadOptions()
  } catch (error) {
    const message = getApiErrorMessage(error)
    pageErrorMessage.value = message
    toast.error(message || 'error')
  } finally {
    submitting.value = false
  }
}

void loadOptions()
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
        <Button variant="outline" size="sm" :disabled="loading || submitting" @click="loadOptions">
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
                <div class="space-y-2">
                  <Label for="withdraw-toko">Toko</Label>
                  <Select v-model="selectedTokoID">
                    <SelectTrigger id="withdraw-toko">
                      <SelectValue placeholder="Pilih toko" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="item in tokos" :key="item.id" :value="String(item.id)">
                        {{ item.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2">
                  <Label for="withdraw-bank">Bank Tujuan</Label>
                  <Select v-model="selectedBankID">
                    <SelectTrigger id="withdraw-bank">
                      <SelectValue placeholder="Pilih bank" />
                    </SelectTrigger>
                    <SelectContent>
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
                      <p class="text-xs text-muted-foreground">Settlement Balance</p>
                      <p class="text-lg font-semibold text-foreground">
                        {{ formatCurrency(selectedToko?.settlement_balance ?? 0) }}
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
                <CardDescription>Selected Settlement Balance</CardDescription>
                <CardTitle class="text-2xl">{{ formatCurrency(selectedToko?.settlement_balance ?? 0) }}</CardTitle>
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
      </template>
    </template>
  </section>
</template>
