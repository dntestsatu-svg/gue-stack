<script setup lang="ts">
import { computed } from 'vue'
import {
  ArrowDownUp,
  BookOpenText,
  Cable,
  CheckCheck,
  Copy,
  Landmark,
  QrCode,
  ReceiptText,
  ShieldCheck,
  TrendingUp,
  TriangleAlert,
  WalletCards,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import PageHeader from '@/components/PageHeader.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { resolveApiBaseURL } from '@/services/http'

type EndpointDoc = {
  audience: string
  method: 'POST' | 'GET' | 'PATCH' | 'DELETE'
  path: string
  auth: string
  purpose: string
  successContract: string
}

type OutcomeDoc = {
  condition: string
  ready: boolean
  message: string
  detail: string
}

const platformFeePercent = 3
const apiBaseURL = computed(() => `${resolveApiBaseURL(import.meta.env.VITE_API_BASE_URL)}/api/v1`)

const productionSteps = [
  'Ambil token toko dari halaman Toko, lalu simpan di backend merchant. Jangan expose ke browser publik.',
  'Atur callback_url toko ke endpoint merchant yang menerima status final transaksi dari project ini.',
  'Generate QRIS lewat endpoint merchant API. Project akan simpan transaksi pending sebelum QR payload dikembalikan.',
  'Terima callback final success, failed, atau expired. Terapkan idempotency merchant berdasarkan trx_id.',
]

const testingSteps = [
  'Route Testing dipakai dari dashboard internal, bukan dari website merchant publik.',
  'Generate QRIS di Testing memakai use case internal yang sama untuk validasi pending transaction dan payload QR.',
  'Callback Readiness mengirim POST probe ke callback_url toko untuk mengecek apakah endpoint merchant benar-benar siap.',
  'Testing lebih ketat daripada callback produksi: harus HTTP 200 + JSON {"success": true}.',
]

const merchantEndpoints: EndpointDoc[] = [
  {
    audience: 'Merchant server',
    method: 'POST',
    path: '/payments/gateway/generate',
    auth: 'Bearer toko token',
    purpose: 'Generate QRIS baru. Jika sukses, project menyimpan transaksi lokal dengan status pending sebelum payload QR dikembalikan.',
    successContract: 'HTTP 200 + payload QRIS + trx_id',
  },
  {
    audience: 'Merchant server',
    method: 'POST',
    path: '/payments/gateway/check-status/{trx_id}',
    auth: 'Bearer toko token',
    purpose: 'Cek status final transaksi. Project akan membaca cache trx_id lebih dulu bila snapshot final sudah tersedia.',
    successContract: 'HTTP 200 + snapshot status final/pending',
  },
]

const testingEndpoints: EndpointDoc[] = [
  {
    audience: 'Dashboard operator',
    method: 'POST',
    path: '/testing/generate-qris',
    auth: 'Session dashboard',
    purpose: 'Generate QRIS test untuk toko yang ada di scope user login dan mengukur server_processing_ms dari backend.',
    successContract: 'HTTP 200 + data, trx_id, expired_at, server_processing_ms',
  },
  {
    audience: 'Dashboard operator',
    method: 'POST',
    path: '/testing/callback-readiness',
    auth: 'Session dashboard',
    purpose: 'Mengirim probe ke callback_url toko, lalu menilai ready/not ready berdasarkan status HTTP, body JSON, dan flag success.',
    successContract: 'HTTP 200 + ready=true + status_code + received_success + latency metrics',
  },
]

const readinessOutcomes: OutcomeDoc[] = [
  {
    condition: 'callback_url belum dikonfigurasi',
    ready: false,
    message: 'API kamu sepertinya belum terintegrasi dengan baik.',
    detail: 'Callback URL toko belum dikonfigurasi.',
  },
  {
    condition: 'Callback URL invalid / tidak bisa dibuat request probe',
    ready: false,
    message: 'API kamu sepertinya belum terintegrasi dengan baik.',
    detail: 'Callback URL tidak valid atau tidak dapat dipakai untuk probe.',
  },
  {
    condition: 'Server merchant timeout / tidak bisa dijangkau',
    ready: false,
    message: 'API kamu sepertinya belum terintegrasi dengan baik.',
    detail: 'Server callback toko tidak dapat dijangkau atau timeout.',
  },
  {
    condition: 'Body response tidak bisa dibaca atau bukan JSON valid',
    ready: false,
    message: 'API kamu sepertinya belum terintegrasi dengan baik.',
    detail: 'Callback URL belum mengembalikan JSON {"success": true}.',
  },
  {
    condition: 'HTTP bukan 200 atau success=false',
    ready: false,
    message: 'API kamu sepertinya belum terintegrasi dengan baik.',
    detail: 'Callback harus merespons HTTP 200 dengan body {"success": true}.',
  },
  {
    condition: 'HTTP 200 dan body {"success": true}',
    ready: true,
    message: 'API kamu sudah ready.',
    detail: 'Callback URL merespons sesuai kontrak integrasi.',
  },
]

const authHeaderSnippet = computed(() => `Authorization: Bearer YOUR_TOKO_TOKEN
Content-Type: application/json
Accept: application/json`)

const generateSnippet = computed(() => `curl -X POST "${apiBaseURL.value}/payments/gateway/generate" \\
  -H "Authorization: Bearer YOUR_TOKO_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "player-001",
    "amount": 100000,
    "expire": 300,
    "custom_ref": "ORDER-001"
  }'`)

const generateSuccessSnippet = `{
  "status": "success",
  "data": {
    "toko_id": 7,
    "toko_name": "Toko Alpha",
    "data": "00020101021226670016COM.NOBUBANK.WWW01189360050300000879140214982601430403470303UMI51440014ID.CO.QRIS.WWW0215ID20262000000000303UMI52045411530336054061000005802ID5920TOKO ALPHA DEMO QRIS6013JAKARTA BARAT6105115606304A1B2",
    "trx_id": "trx-123456",
    "expired_at": 1764205500,
    "server_processing_ms": 148
  }
}`

const statusSnippet = computed(() => `curl -X POST "${apiBaseURL.value}/payments/gateway/check-status/TRX_ID_ANDA" \\
  -H "Authorization: Bearer YOUR_TOKO_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{}'`)

const testingProbeHeadersSnippet = `Content-Type: application/json
Accept: application/json
User-Agent: gue-testing-probe/1.0
X-GUE-Testing: true`

const testingProbePayloadSnippet = `{
  "type": "integration_check",
  "source": "gue",
  "toko_id": 7,
  "toko_name": "Toko Alpha",
  "timestamp": "2026-03-25T10:00:00Z"
}`
const testingReadySuccessSnippet = `{
  "status": "success",
  "data": {
    "toko_id": 7,
    "toko_name": "Toko Alpha",
    "callback_url": "https://merchant.example.com/callback",
    "ready": true,
    "message": "API kamu sudah ready.",
    "detail": "Callback URL merespons sesuai kontrak integrasi.",
    "status_code": 200,
    "received_success": true,
    "response_excerpt": "{\\"success\\":true}",
    "callback_latency_ms": 92,
    "server_processing_ms": 97
  }
}`

const testingReadyFailureSnippet = `{
  "status": "success",
  "data": {
    "toko_id": 7,
    "toko_name": "Toko Alpha",
    "callback_url": "https://merchant.example.com/callback",
    "ready": false,
    "message": "API kamu sepertinya belum terintegrasi dengan baik.",
    "detail": "Callback harus merespons HTTP 200 dengan body {\\"success\\": true}.",
    "status_code": 500,
    "received_success": false,
    "response_excerpt": "{\\"success\\":false}",
    "callback_latency_ms": 104,
    "server_processing_ms": 111
  }
}`

const callbackPayloadSnippet = `{
  "amount": 100000,
  "terminal_id": "TERM-001",
  "merchant_id": "TOKEN_TOKO_USER",
  "trx_id": "trx-123456",
  "rrn": "rrn-123456",
  "custom_ref": "ORDER-001",
  "vendor": "motpay",
  "status": "success",
  "created_at": "2026-03-22T10:00:00Z",
  "finish_at": "2026-03-22T10:00:12Z"
}`

const callbackAckSnippet = `HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true
}`

const financialFormulaSnippet = `Deposit success:
gross_amount            = amount dari external API
platform_fee_project    = round(gross_amount x ${platformFeePercent}%)
net_credit_to_pending   = gross_amount - platform_fee_project

Manual settlement:
pending_balance -= settlement_amount
settle_balance  += settlement_amount

Withdraw:
settle_balance -= (withdraw_amount + fee_withdrawal_external)

Catatan:
- fee_withdrawal_external bukan profit project
- profit project hanya berasal dari platform_fee transaksi deposit success`

const merchantChecklist = [
  'Pisahkan kontrak merchant API dan dashboard testing lab. Jangan menganggap readiness probe sama dengan callback produksi.',
  'Simpan bearer token toko di backend merchant, bukan di frontend browser publik.',
  'Saat menerima callback final, lakukan idempotency berdasarkan trx_id agar callback ulang tidak memproses order dua kali.',
  'Gunakan custom_ref untuk menghubungkan transaksi gateway dengan order internal merchant.',
  'Balas callback final dengan 2xx. Untuk observability yang konsisten, tetap disarankan body {"success": true}.',
]

const copyText = async (value: string, label: string) => {
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
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Merchant Integration"
      title="Dokumentasi API"
      description="Panduan implementasi merchant API, dashboard testing lab, callback contract, dan transparansi finansial agar integrasi tidak ambigu."
    >
      <template #actions>
        <a href="#testing-lab" class="app-header-control inline-flex items-center justify-center text-sm font-medium no-underline">
          Buka Section Testing
        </a>
        <Button variant="outline" size="sm" @click="copyText(apiBaseURL, 'Base URL API')">
          <Copy class="mr-2 h-4 w-4" />
          Copy Base URL
        </Button>
      </template>
    </PageHeader>

    <Alert class="app-panel">
      <TriangleAlert class="h-4 w-4" />
      <AlertTitle>Jangan campur dua kontrak callback</AlertTitle>
      <AlertDescription>
        <span class="font-medium text-foreground">Callback produksi</span> dianggap delivered jika merchant membalas
        <span class="font-medium text-foreground">2xx</span>. Sementara
        <span class="font-medium text-foreground">Callback Readiness</span> di halaman Testing jauh lebih ketat:
        <span class="font-medium text-foreground">HTTP 200 + JSON {"success": true}</span>.
      </AlertDescription>
    </Alert>

    <div class="grid gap-4 xl:grid-cols-[minmax(0,1.12fr)_minmax(0,0.88fr)]">
      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-start gap-3">
            <div class="docs-metric-icon"><BookOpenText class="h-4 w-4" /></div>
            <div class="space-y-1">
              <CardTitle>Production Integration</CardTitle>
              <CardDescription>
                Jalur ini dipakai merchant website untuk transaksi live: generate QRIS, check status, dan menerima callback final.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-3">
          <div v-for="step in productionSteps" :key="step" class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <p class="text-sm text-muted-foreground">{{ step }}</p>
          </div>
        </CardContent>
      </Card>

      <Card id="testing-lab" class="app-panel">
        <CardHeader>
          <div class="flex items-start gap-3">
            <div class="docs-metric-icon"><ShieldCheck class="h-4 w-4" /></div>
            <div class="space-y-1">
              <CardTitle>Testing Lab untuk Callback URL</CardTitle>
              <CardDescription>
                Jalur ini dipakai operator dashboard untuk menguji toko, probe callback_url, dan membaca alasan sukses atau gagal dengan detail.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-3">
          <div v-for="step in testingSteps" :key="step" class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <p class="text-sm text-muted-foreground">{{ step }}</p>
          </div>
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      <a href="#merchant-api" class="app-tone-card block space-y-2 no-underline">
        <p class="dashboard-eyebrow">01</p>
        <p class="text-sm font-semibold text-foreground">Merchant API</p>
        <p class="text-sm text-muted-foreground">Endpoint untuk website merchant.</p>
      </a>
      <a href="#testing-contract" class="app-tone-card block space-y-2 no-underline">
        <p class="dashboard-eyebrow">02</p>
        <p class="text-sm font-semibold text-foreground">Testing Contract</p>
        <p class="text-sm text-muted-foreground">Probe payload, sukses, dan gagal.</p>
      </a>
      <a href="#merchant-callback" class="app-tone-card block space-y-2 no-underline">
        <p class="dashboard-eyebrow">03</p>
        <p class="text-sm font-semibold text-foreground">Final Callback</p>
        <p class="text-sm text-muted-foreground">Payload final yang dikirim ke merchant.</p>
      </a>
      <a href="#financial-rules" class="app-tone-card block space-y-2 no-underline">
        <p class="dashboard-eyebrow">04</p>
        <p class="text-sm font-semibold text-foreground">Financial Rules</p>
        <p class="text-sm text-muted-foreground">Alur angka dan saldo yang berlaku.</p>
      </a>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-start gap-3">
            <div class="docs-metric-icon"><Cable class="h-4 w-4" /></div>
            <div class="space-y-1">
              <CardTitle>Callback Contract: Production vs Testing</CardTitle>
              <CardDescription>
                Section ini menjelaskan perbedaan kontrak yang paling sering membuat integrasi merchant gagal dibaca dengan benar.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="docs-list-item">
            <Cable class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <div>
              <p class="text-sm font-medium text-foreground">Final callback produksi</p>
              <p class="text-sm text-muted-foreground">
                Backend project mengirim callback final ke merchant setelah webhook tervalidasi. Saat ini delivery dianggap sukses jika merchant
                merespons <code>2xx</code>. Body response tidak diparsing untuk menandai delivery berhasil.
              </p>
            </div>
          </div>
          <div class="docs-list-item">
            <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <div>
              <p class="text-sm font-medium text-foreground">Callback Readiness di halaman Testing</p>
              <p class="text-sm text-muted-foreground">
                Probe readiness hanya dianggap <code>ready=true</code> jika merchant merespons <code>HTTP 200</code> dan body JSON
                <code>{"success": true}</code>. Jika salah satu tidak terpenuhi, hasilnya tetap <code>ready=false</code>.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Merchant Checklist</CardTitle>
          <CardDescription>
            Guardrails yang sebaiknya dipenuhi merchant sebelum go-live agar dokumentasi ini benar-benar bisa dieksekusi.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <div v-for="item in merchantChecklist" :key="item" class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <p class="text-sm text-muted-foreground">{{ item }}</p>
          </div>
        </CardContent>
      </Card>
    </div>

    <div id="merchant-api" class="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
      <Card class="app-panel docs-table-card">
        <CardHeader>
          <CardTitle>Merchant Endpoint Catalog</CardTitle>
          <CardDescription>
            Endpoint ini dipanggil dari backend merchant dan semuanya berada di bawah prefix <code>/api/v1</code>.
          </CardDescription>
        </CardHeader>
        <CardContent class="app-table-shell">
          <Table class="app-data-table docs-table-tight">
            <TableHeader>
              <TableRow>
                <TableHead>Audience</TableHead>
                <TableHead>Method</TableHead>
                <TableHead>Path</TableHead>
                <TableHead>Auth</TableHead>
                <TableHead>Purpose</TableHead>
                <TableHead>Success</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="endpoint in merchantEndpoints" :key="`${endpoint.method}:${endpoint.path}`">
                <TableCell>{{ endpoint.audience }}</TableCell>
                <TableCell>
                  <Badge>{{ endpoint.method }}</Badge>
                </TableCell>
                <TableCell><code>{{ endpoint.path }}</code></TableCell>
                <TableCell>{{ endpoint.auth }}</TableCell>
                <TableCell class="whitespace-normal">{{ endpoint.purpose }}</TableCell>
                <TableCell class="whitespace-normal">{{ endpoint.successContract }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Authorization Header</CardTitle>
          <CardDescription>
            Website merchant memanggil API production dengan Bearer token toko, bukan credential vendor.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="docs-code-block">
            <pre>{{ authHeaderSnippet }}</pre>
          </div>
          <Button variant="outline" size="sm" @click="copyText(authHeaderSnippet, 'Header Authorization')">
            <Copy class="mr-2 h-4 w-4" />
            Copy Header
          </Button>
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-center justify-between gap-3">
            <div class="space-y-1">
              <CardTitle>Generate QRIS Example</CardTitle>
              <CardDescription>
                Request merchant untuk membuat transaksi pending dan mendapatkan payload QRIS.
              </CardDescription>
            </div>
            <Button variant="ghost" size="icon" @click="copyText(generateSnippet, 'Snippet Generate QRIS')">
              <Copy class="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div class="docs-code-block">
            <pre>{{ generateSnippet }}</pre>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-center justify-between gap-3">
            <div class="space-y-1">
              <CardTitle>Generate QRIS Success Response</CardTitle>
              <CardDescription>
                Bentuk respons sukses yang dipakai dashboard testing dan mudah dijadikan acuan oleh tim merchant saat membaca output backend.
              </CardDescription>
            </div>
            <Button variant="ghost" size="icon" @click="copyText(generateSuccessSnippet, 'Contoh response generate sukses')">
              <Copy class="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div class="docs-code-block">
            <pre>{{ generateSuccessSnippet }}</pre>
          </div>
        </CardContent>
      </Card>
    </div>

    <Card class="app-panel">
      <CardHeader>
        <div class="flex items-center gap-2">
          <QrCode class="h-5 w-5 text-(--brand)" />
          <div>
            <CardTitle>Check Status Example</CardTitle>
            <CardDescription>
              Gunakan route ini jika merchant ingin melakukan polling manual. Bila snapshot final sudah ada, project akan membaca cache <code>trx_id</code> lebih dulu.
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div class="docs-code-block">
          <pre>{{ statusSnippet }}</pre>
        </div>
      </CardContent>
    </Card>

    <div id="testing-contract" class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <Card class="app-panel docs-table-card">
        <CardHeader>
          <CardTitle>Testing Endpoint Catalog</CardTitle>
          <CardDescription>
            Endpoint internal ini dipakai oleh halaman Testing di dashboard untuk memvalidasi toko, generate QRIS test, dan mengecek kesiapan callback_url.
          </CardDescription>
        </CardHeader>
        <CardContent class="app-table-shell">
          <Table class="app-data-table docs-table-tight">
            <TableHeader>
              <TableRow>
                <TableHead>Audience</TableHead>
                <TableHead>Method</TableHead>
                <TableHead>Path</TableHead>
                <TableHead>Auth</TableHead>
                <TableHead>Purpose</TableHead>
                <TableHead>Success</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="endpoint in testingEndpoints" :key="`${endpoint.method}:${endpoint.path}`">
                <TableCell>{{ endpoint.audience }}</TableCell>
                <TableCell>
                  <Badge>{{ endpoint.method }}</Badge>
                </TableCell>
                <TableCell><code>{{ endpoint.path }}</code></TableCell>
                <TableCell>{{ endpoint.auth }}</TableCell>
                <TableCell class="whitespace-normal">{{ endpoint.purpose }}</TableCell>
                <TableCell class="whitespace-normal">{{ endpoint.successContract }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Probe Payload yang Dikirim ke callback_url</CardTitle>
          <CardDescription>
            Saat operator menekan <code>Check Callback Readiness</code>, backend akan mengirim POST probe berikut ke <code>callback_url</code> toko.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="docs-code-block">
            <pre>{{ testingProbeHeadersSnippet }}</pre>
          </div>
          <div class="docs-code-block">
            <pre>{{ testingProbePayloadSnippet }}</pre>
          </div>
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Response Readiness: Success</CardTitle>
          <CardDescription>
            Inilah bentuk respons sukses dari route testing saat callback_url merespons tepat sesuai kontrak readiness.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="docs-code-block">
            <pre>{{ testingReadySuccessSnippet }}</pre>
          </div>
          <div class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <p class="text-sm text-muted-foreground">
              Success readiness mensyaratkan <code>HTTP 200</code>, JSON valid, dan <code>received_success=true</code>.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Response Readiness: Failure</CardTitle>
          <CardDescription>
            Jika respons merchant tidak sesuai, backend tetap mengembalikan payload terstruktur agar operator bisa langsung tahu penyebabnya.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="docs-code-block">
            <pre>{{ testingReadyFailureSnippet }}</pre>
          </div>
          <div class="docs-list-item">
            <TriangleAlert class="mt-0.5 h-4 w-4 shrink-0 text-(--danger)" />
            <p class="text-sm text-muted-foreground">
              Kasus yang paling sering: endpoint membalas <code>500</code>, body bukan JSON, atau body JSON tetapi <code>success=false</code>.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>

    <Card class="app-panel">
      <CardHeader>
        <CardTitle>Bagaimana Jika Responsenya Tidak Sukses?</CardTitle>
        <CardDescription>
          Matriks ini menjelaskan apa yang dianggap sukses atau gagal oleh backend saat menjalankan Callback Readiness dari halaman Testing.
        </CardDescription>
      </CardHeader>
      <CardContent class="app-table-shell">
        <Table class="app-data-table docs-table-tight">
          <TableHeader>
            <TableRow>
              <TableHead>Condition</TableHead>
              <TableHead>Ready</TableHead>
              <TableHead>Message</TableHead>
              <TableHead>Detail</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in readinessOutcomes" :key="row.condition">
              <TableCell class="whitespace-normal">{{ row.condition }}</TableCell>
              <TableCell>
                <Badge :variant="row.ready ? 'default' : 'secondary'">
                  {{ row.ready ? 'true' : 'false' }}
                </Badge>
              </TableCell>
              <TableCell class="whitespace-normal">{{ row.message }}</TableCell>
              <TableCell class="whitespace-normal">{{ row.detail }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Card id="merchant-callback" class="app-panel">
      <CardHeader>
        <div class="flex items-center gap-2">
          <Cable class="h-5 w-5 text-(--brand)" />
          <div>
            <CardTitle>Callback Payload ke Merchant Website</CardTitle>
            <CardDescription>
              Setelah webhook external API tervalidasi dan transaksi difinalisasi, project akan mengirim payload final berikut ke <code>callback_url</code> merchant.
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="docs-code-block">
          <pre>{{ callbackPayloadSnippet }}</pre>
        </div>
        <div class="grid gap-4 lg:grid-cols-2">
          <div class="docs-list-item">
            <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <p class="text-sm text-muted-foreground">
              Field <code>merchant_id</code> pada callback merchant berisi <span class="font-medium text-foreground">token toko</span>, bukan merchant UUID vendor.
            </p>
          </div>
          <div class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <p class="text-sm text-muted-foreground">
              Untuk delivery produksi, backend saat ini menandai sukses jika merchant membalas <code>2xx</code>. Untuk observability yang konsisten, tetap disarankan body <code>{"success": true}</code>.
            </p>
          </div>
        </div>
        <Separator />
        <div class="grid gap-4 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]">
          <div class="space-y-3">
            <p class="text-sm font-medium text-foreground">HTTP response merchant yang direkomendasikan</p>
            <div class="docs-code-block">
              <pre>{{ callbackAckSnippet }}</pre>
            </div>
          </div>
          <div class="space-y-3">
            <div class="docs-list-item">
              <ReceiptText class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
              <p class="text-sm text-muted-foreground">
                Walaupun delivery produksi hanya memeriksa <code>2xx</code>, respons <code>{"success": true}</code> tetap direkomendasikan agar log merchant dan hasil probe dashboard konsisten.
              </p>
            </div>
            <div class="docs-list-item">
              <TriangleAlert class="mt-0.5 h-4 w-4 shrink-0 text-(--warning)" />
              <p class="text-sm text-muted-foreground">
                Jika merchant membalas <code>4xx/5xx</code>, project akan menganggap callback ditolak dan delivery tidak ditandai sukses.
              </p>
            </div>
            <Button variant="outline" size="sm" @click="copyText(callbackAckSnippet, 'Response ACK merchant')">
              <Copy class="mr-2 h-4 w-4" />
              Copy ACK Example
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <div class="grid gap-4 lg:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]">
      <Card id="financial-rules" class="app-panel">
        <CardHeader>
          <div class="flex items-start gap-3">
            <div class="docs-metric-icon"><WalletCards class="h-4 w-4" /></div>
            <div class="space-y-1">
              <CardTitle>Financial Rules yang Berlaku</CardTitle>
              <CardDescription>
                Ringkasan aturan saldo, fee, dan arus nilai yang dipakai project agar merchant tidak salah membaca pending balance, settle balance, dan profit project.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="docs-code-block">
            <pre>{{ financialFormulaSnippet }}</pre>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <div class="app-tone-card space-y-2">
              <div class="flex items-center gap-2 text-foreground">
                <TrendingUp class="h-4 w-4 text-(--success)" />
                <p class="text-sm font-semibold">Project Profit</p>
              </div>
              <p class="text-2xl font-semibold text-foreground">{{ platformFeePercent }}%</p>
              <p class="dashboard-eyebrow">Deposit fee</p>
              <p class="text-sm text-muted-foreground">Hanya berasal dari deposit success final. Fee withdraw vendor bukan profit project.</p>
            </div>
            <div class="app-tone-card space-y-2">
              <div class="flex items-center gap-2 text-foreground">
                <ArrowDownUp class="h-4 w-4 text-(--brand)" />
                <p class="text-sm font-semibold">Pending ke Settle</p>
              </div>
              <p class="text-sm text-muted-foreground">Dana masuk ke pending lebih dulu. Perpindahan ke settle dilakukan lewat proses settlement manual di dashboard.</p>
            </div>
            <div class="app-tone-card space-y-2">
              <div class="flex items-center gap-2 text-foreground">
                <Landmark class="h-4 w-4 text-(--warning)" />
                <p class="text-sm font-semibold">Withdraw External Fee</p>
              </div>
              <p class="text-sm text-muted-foreground">Biaya transfer vendor mengurangi settle balance toko saat withdraw diproses, bukan masuk ke profit project.</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-start gap-3">
            <div class="docs-metric-icon"><ReceiptText class="h-4 w-4" /></div>
            <div class="space-y-1">
              <CardTitle>Halaman Testing Menjelaskan Apa?</CardTitle>
              <CardDescription>
                Route Testing bukan dokumentasi tambahan semata. Halaman itu menunjukkan bukti operasional apakah toko benar-benar siap dipakai merchant live.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="docs-list-item">
            <QrCode class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <p class="text-sm text-muted-foreground">
              <span class="font-medium text-foreground">Generate QRIS</span> menunjukkan transaksi test berhasil dibuat, lengkap dengan <code>trx_id</code>, payload QR, <code>expired_at</code>, dan <code>server_processing_ms</code>.
            </p>
          </div>
          <div class="docs-list-item">
            <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <p class="text-sm text-muted-foreground">
              <span class="font-medium text-foreground">Callback Readiness</span> menjelaskan apakah callback_url merchant bisa dijangkau dan sudah membalas sesuai kontrak readiness yang ketat.
            </p>
          </div>
          <div class="docs-list-item">
            <TriangleAlert class="mt-0.5 h-4 w-4 shrink-0 text-(--warning)" />
            <p class="text-sm text-muted-foreground">
              Jika hasilnya gagal, operator harus membaca kombinasi <code>message</code>, <code>detail</code>, <code>status_code</code>, <code>received_success</code>, dan <code>response_excerpt</code> untuk tahu titik rusaknya.
            </p>
          </div>
          <div class="docs-list-item">
            <WalletCards class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <p class="text-sm text-muted-foreground">
              Testing tidak mengubah kontrak callback produksi. Ia dipakai sebagai lab internal supaya merchant dan operator punya bukti kesiapan sebelum go-live.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  </section>
</template>
