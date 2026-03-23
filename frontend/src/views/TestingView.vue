<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { CheckCircle2, Copy, QrCode, RefreshCcw, ShieldCheck, TriangleAlert } from 'lucide-vue-next'
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
import * as testingApi from '@/services/testing'
import * as tokoApi from '@/services/toko'
import type { TestingCallbackReadinessResult, TestingGenerateQrisResult, TokoItem } from '@/services/types'

const { formatCurrency, formatDateMedium } = useFormatters()

const tokos = ref<TokoItem[]>([])
const loadingTokos = ref(false)
const pageErrorMessage = ref('')
const lastUpdated = ref('')

const selectedTokoID = ref('')

const generateLoading = ref(false)
const readinessLoading = ref(false)
const generateResult = ref<TestingGenerateQrisResult | null>(null)
const readinessResult = ref<TestingCallbackReadinessResult | null>(null)

const generateForm = reactive({
  username: '',
  amount: '10000',
  expire: '300',
  customRef: '',
})

const selectedToko = computed(() =>
  tokos.value.find((item) => String(item.id) === selectedTokoID.value) ?? null,
)

const canRunTesting = computed(() => selectedToko.value !== null)
const callbackConfigured = computed(() => Boolean(selectedToko.value?.callback_url))

const copyToClipboard = async (value: string, label: string) => {
  if (typeof window === 'undefined' || !window.navigator?.clipboard) {
    toast.error('Clipboard tidak tersedia di browser ini')
    return
  }

  try {
    await window.navigator.clipboard.writeText(value)
    toast.success(`${label} berhasil dicopy`)
  } catch {
    toast.error(`Gagal menyalin ${label.toLowerCase()}`)
  }
}

const formatCallbackStatus = (ready: boolean) => (ready ? 'Ready' : 'Not Ready')

const formatUnixTime = (value?: number) => {
  if (!value) {
    return '-'
  }

  let unixSeconds = value
  if (unixSeconds >= 1_000_000_000_000) {
    unixSeconds = Math.floor(unixSeconds / 1000)
  } else if (unixSeconds < 946684800) {
    unixSeconds = Math.floor(Date.now() / 1000) + unixSeconds
  }

  return formatDateMedium(new Date(unixSeconds * 1000).toISOString())
}

const loadTokos = async () => {
  loadingTokos.value = true
  pageErrorMessage.value = ''
  try {
    const result = await tokoApi.fetchTokos()
    tokos.value = result
    if (!selectedTokoID.value && result.length > 0) {
      selectedTokoID.value = String(result[0].id)
    }
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    pageErrorMessage.value = getApiErrorMessage(error)
  } finally {
    loadingTokos.value = false
  }
}

const submitGenerateQris = async () => {
  if (!selectedToko.value) {
    toast.error('Pilih toko terlebih dahulu')
    return
  }

  generateLoading.value = true
  pageErrorMessage.value = ''
  try {
    const amount = Number(generateForm.amount)
    const expire = generateForm.expire.trim() === '' ? undefined : Number(generateForm.expire)
    generateResult.value = await testingApi.generateQris({
      toko_id: selectedToko.value.id,
      username: generateForm.username.trim(),
      amount,
      expire,
      custom_ref: generateForm.customRef.trim() || undefined,
    })
    toast.success('QRIS test berhasil digenerate')
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    const message = getApiErrorMessage(error)
    pageErrorMessage.value = message
    toast.error(message)
  } finally {
    generateLoading.value = false
  }
}

const submitCheckCallback = async () => {
  if (!selectedToko.value) {
    toast.error('Pilih toko terlebih dahulu')
    return
  }

  readinessLoading.value = true
  pageErrorMessage.value = ''
  try {
    readinessResult.value = await testingApi.checkCallbackReadiness({
      toko_id: selectedToko.value.id,
    })
    if (readinessResult.value.ready) {
      toast.success(readinessResult.value.message)
    } else {
      toast.error(readinessResult.value.message)
    }
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    const message = getApiErrorMessage(error)
    pageErrorMessage.value = message
    toast.error(message)
  } finally {
    readinessLoading.value = false
  }
}

void loadTokos()
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Integration Lab"
      title="Testing Toko"
      description="Generate QRIS untuk sandbox operasional dan validasi readiness callback_url merchant."
      :updated-at="lastUpdated"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="loadingTokos" @click="loadTokos">
          <RefreshCcw class="mr-2 h-4 w-4" />
          {{ loadingTokos ? 'Refreshing...' : 'Refresh Toko' }}
        </Button>
      </template>
    </PageHeader>

    <Alert v-if="pageErrorMessage" variant="destructive">
      <TriangleAlert class="h-4 w-4" />
      <AlertTitle>Testing Request Failed</AlertTitle>
      <AlertDescription>{{ pageErrorMessage }}</AlertDescription>
    </Alert>

    <div v-if="loadingTokos && tokos.length === 0" class="space-y-4">
      <Card class="app-panel">
        <CardContent class="space-y-3 p-6">
          <Skeleton class="h-6 w-48" />
          <Skeleton class="h-11 w-full" />
        </CardContent>
      </Card>
      <div class="grid gap-4 xl:grid-cols-2">
        <Card class="app-panel">
          <CardContent class="space-y-3 p-6">
            <Skeleton class="h-6 w-52" />
            <Skeleton class="h-11 w-full" />
            <Skeleton class="h-11 w-full" />
            <Skeleton class="h-32 w-full" />
          </CardContent>
        </Card>
        <Card class="app-panel">
          <CardContent class="space-y-3 p-6">
            <Skeleton class="h-6 w-52" />
            <Skeleton class="h-11 w-full" />
            <Skeleton class="h-28 w-full" />
          </CardContent>
        </Card>
      </div>
    </div>

    <template v-else>
      <Card v-if="tokos.length === 0" class="app-panel">
        <CardContent class="p-6">
          <EmptyState
            title="Belum Ada Toko yang Bisa Diuji"
            description="Testing page hanya menampilkan toko yang memang berada dalam scope akun yang sedang login."
          />
        </CardContent>
      </Card>

      <template v-else>
        <Card class="app-panel">
          <CardHeader>
            <CardTitle>Target Toko</CardTitle>
            <CardDescription>Pilih toko yang akan dipakai untuk generate QRIS dan callback readiness check.</CardDescription>
          </CardHeader>
          <CardContent class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
            <div class="space-y-2">
              <Label for="testing-toko">Toko</Label>
              <Select v-model="selectedTokoID">
                <SelectTrigger id="testing-toko">
                  <SelectValue placeholder="Pilih toko" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="item in tokos" :key="item.id" :value="String(item.id)">
                    {{ item.name }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{{ selectedToko?.charge ?? 0 }}% charge</Badge>
              <Badge :variant="callbackConfigured ? 'default' : 'secondary'">
                {{ callbackConfigured ? 'Callback Configured' : 'No Callback URL' }}
              </Badge>
            </div>

            <div class="app-tone-card lg:col-span-2">
              <p class="text-sm font-medium text-foreground">Callback URL</p>
              <p class="mt-1 break-all text-sm text-muted-foreground">
                {{ selectedToko?.callback_url || 'Belum ada callback URL yang tersimpan untuk toko ini.' }}
              </p>
            </div>
          </CardContent>
        </Card>

        <div class="grid gap-4 xl:grid-cols-2">
          <Card class="app-panel">
            <CardHeader>
              <CardTitle>Generate QRIS</CardTitle>
              <CardDescription>Menembak use case payment bridge internal untuk membuat transaksi QRIS test.</CardDescription>
            </CardHeader>
            <CardContent class="space-y-4">
              <div class="grid gap-4 md:grid-cols-2">
                <div class="space-y-2 md:col-span-2">
                  <Label for="testing-username">Username / Player</Label>
                  <Input id="testing-username" v-model="generateForm.username" placeholder="player-demo-01" />
                </div>
                <div class="space-y-2">
                  <Label for="testing-amount">Amount</Label>
                  <Input id="testing-amount" v-model="generateForm.amount" type="number" min="10000" step="1000" />
                </div>
                <div class="space-y-2">
                  <Label for="testing-expire">Expire (detik)</Label>
                  <Input id="testing-expire" v-model="generateForm.expire" type="number" min="30" step="30" />
                </div>
                <div class="space-y-2 md:col-span-2">
                  <Label for="testing-custom-ref">Custom Ref (opsional)</Label>
                  <Input id="testing-custom-ref" v-model="generateForm.customRef" placeholder="ORDER001" />
                </div>
              </div>

              <div class="flex justify-end">
                <Button :disabled="generateLoading || !canRunTesting" @click="submitGenerateQris">
                  <Spinner v-if="generateLoading" class="mr-2" />
                  <QrCode v-else class="mr-2 h-4 w-4" />
                  {{ generateLoading ? 'Generating...' : 'Generate QRIS' }}
                </Button>
              </div>

              <div v-if="generateResult" class="app-tone-card">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-foreground">{{ generateResult.toko_name }}</p>
                    <p class="text-xs text-muted-foreground">Trx ID: {{ generateResult.trx_id }}</p>
                  </div>
                  <Badge variant="outline">{{ generateResult.server_processing_ms }} ms server</Badge>
                </div>

                <div class="mt-4 grid gap-3 md:grid-cols-2">
                  <div class="rounded-lg border bg-background p-3">
                    <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">Amount</p>
                    <p class="mt-1 text-sm font-medium text-foreground">{{ formatCurrency(Number(generateForm.amount || 0)) }}</p>
                  </div>
                  <div class="rounded-lg border bg-background p-3">
                    <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">Expired At</p>
                    <p class="mt-1 text-sm font-medium text-foreground">{{ formatUnixTime(generateResult.expired_at) }}</p>
                  </div>
                </div>

                <div class="mt-4 space-y-2">
                  <div class="flex items-center justify-between gap-3">
                    <p class="text-sm font-medium text-foreground">QR Payload</p>
                    <Button size="sm" variant="outline" @click="copyToClipboard(generateResult.data, 'QR payload')">
                      <Copy class="mr-2 h-4 w-4" />
                      Copy
                    </Button>
                  </div>
                  <code class="block rounded-lg border bg-background px-3 py-2 text-xs break-all">{{ generateResult.data }}</code>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card class="app-panel">
            <CardHeader>
              <CardTitle>Callback Readiness</CardTitle>
              <CardDescription>
                Backend akan mengirim probe ke <code>callback_url</code> toko dan memverifikasi kontrak HTTP 200 + JSON
                <code>{ "success": true }</code>.
              </CardDescription>
            </CardHeader>
            <CardContent class="space-y-4">
              <div class="app-tone-card">
                <p class="text-sm text-muted-foreground">
                  Jika response tidak sesuai kontrak, user akan diberi tahu bahwa API merchant belum terintegrasi dengan baik.
                </p>
              </div>

              <div class="flex justify-end">
                <Button variant="outline" :disabled="readinessLoading || !canRunTesting" @click="submitCheckCallback">
                  <Spinner v-if="readinessLoading" class="mr-2" />
                  <ShieldCheck v-else class="mr-2 h-4 w-4" />
                  {{ readinessLoading ? 'Checking...' : 'Check Callback Readiness' }}
                </Button>
              </div>

              <Alert
                v-if="readinessResult"
                :variant="readinessResult.ready ? 'default' : 'destructive'"
              >
                <CheckCircle2 v-if="readinessResult.ready" class="h-4 w-4" />
                <TriangleAlert v-else class="h-4 w-4" />
                <AlertTitle>{{ readinessResult.message }}</AlertTitle>
                <AlertDescription>
                  {{ readinessResult.detail }}
                </AlertDescription>
              </Alert>

              <div v-if="readinessResult" class="grid gap-3 md:grid-cols-2">
                <div class="app-info-tile">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">Callback Latency</p>
                  <p class="mt-2 text-2xl font-semibold text-foreground">{{ readinessResult.callback_latency_ms }} ms</p>
                </div>
                <div class="app-info-tile">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">Go Server Processing</p>
                  <p class="mt-2 text-2xl font-semibold text-foreground">{{ readinessResult.server_processing_ms }} ms</p>
                </div>
                <div class="app-info-tile">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">HTTP Status</p>
                  <p class="mt-2 text-2xl font-semibold text-foreground">{{ readinessResult.status_code || '-' }}</p>
                </div>
                <div class="app-info-tile">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">Probe Result</p>
                  <div class="mt-2 flex items-center gap-2">
                    <Badge :variant="readinessResult.ready ? 'default' : 'secondary'">
                      {{ formatCallbackStatus(readinessResult.ready) }}
                    </Badge>
                    <span class="text-sm text-muted-foreground">success flag: {{ readinessResult.received_success ? 'true' : 'false' }}</span>
                  </div>
                </div>
              </div>

              <div v-if="readinessResult?.response_excerpt" class="space-y-2">
                <p class="text-sm font-medium text-foreground">Response Excerpt</p>
                <code class="block rounded-lg border bg-(--background-muted) px-3 py-2 text-xs break-all">
                  {{ readinessResult.response_excerpt }}
                </code>
              </div>
            </CardContent>
          </Card>
        </div>
      </template>
    </template>
  </section>
</template>
