<script setup lang="ts">
import { computed } from 'vue'
import {
  ArrowDownUp,
  BookOpenText,
  Cable,
  CheckCheck,
  Copy,
  Landmark,
  ReceiptText,
  ShieldCheck,
  TrendingUp,
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
  method: 'POST' | 'GET' | 'PATCH' | 'DELETE'
  path: string
  auth: string
  purpose: string
}

type StepDoc = {
  title: string
  description: string
}

type InsightDoc = {
  title: string
  description: string
}

const platformFeePercent = 3
const apiBaseURL = computed(() => `${resolveApiBaseURL(import.meta.env.VITE_API_BASE_URL)}/api/v1`)

const quickStartSteps: StepDoc[] = [
  {
    title: '1. Ambil Token Toko',
    description: 'Ambil token toko, simpan di backend merchant, lalu kirim sebagai Bearer token saat memanggil API.',
  },
  {
    title: '2. Simpan Callback URL',
    description: 'Isi callback_url toko ke endpoint merchant yang menerima status final transaksi dari project ini.',
  },
  {
    title: '3. Generate QRIS',
    description: 'Panggil endpoint generate. Project meneruskan ke external API, menyimpan pending, lalu mengembalikan QR payload.',
  },
  {
    title: '4. Terima Callback Final',
    description: 'Saat external API mengirim webhook final, project memprosesnya secara idempotent lalu callback ke merchant.',
  },
]

const endpointInsights: InsightDoc[] = [
  {
    title: 'Request Prefix',
    description: 'Semua endpoint merchant berada di bawah /api/v1 dan menerima request JSON.',
  },
  {
    title: 'Auth Contract',
    description: 'Semua request merchant memakai Bearer token toko. Dan methodnya POST.',
  },
  {
    title: 'Final Status',
    description: 'Status final yang perlu ditangani merchant hanya success, failed, atau expired.',
  },
]

const merchantEndpoints: EndpointDoc[] = [
  {
    method: 'POST',
    path: '/payments/gateway/generate',
    auth: 'Bearer toko token',
    purpose: 'Generate QRIS baru. Jika sukses, transaksi lokal disimpan dengan status pending.',
  },
  {
    method: 'POST',
    path: '/payments/gateway/check-status/{trx_id}',
    auth: 'Bearer toko token',
    purpose: 'Cek status transaksi per trx_id. Project akan memanfaatkan cache trx_id bila snapshot final sudah tersedia.',
  },
]

const operationalNotes: StepDoc[] = [
  {
    title: 'Pending Balance',
    description: 'Setiap deposit success akan menambah pending balance toko sebesar amount - platform_fee.',
  },
  {
    title: 'Settle Balance',
    description: 'Settle balance tidak muncul otomatis dari deposit. Saldo ini diisi lewat settlement manual developer yang memindahkan pending -> settle.',
  },
  {
    title: 'Withdraw Source',
    description: 'Withdraw hanya boleh memakai settle balance. Pending balance tidak dipakai langsung untuk transfer.',
  },
  {
    title: 'Withdraw Fee',
    description: 'Fee withdraw external tidak dihitung sebagai profit project. Fee ini tetap membebani saldo settle toko pada saat withdraw.',
  },
  {
    title: 'Status Lifecycle',
    description: 'Merchant cukup memperlakukan success, failed, dan expired sebagai status final transaksi.',
  },
  {
    title: '20 Minute Expiry',
    description: 'Transaksi pending yang melewati 20 menit akan di-expire otomatis oleh scheduler project.',
  },
  {
    title: 'Check-Status Cache',
    description: 'Check status dapat membaca snapshot final dari cache trx_id agar tidak selalu menembak external API.',
  },
  {
    title: 'Callback Contract',
    description: 'Endpoint callback merchant idealnya membalas HTTP 200 dengan body {"success": true}.',
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

const statusSnippet = computed(() => `curl -X POST "${apiBaseURL.value}/payments/gateway/check-status/TRX_ID_ANDA" \\
  -H "Authorization: Bearer YOUR_TOKO_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{}'`)

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

const errorResponseSnippet = `{
  "status": "error",
  "message": "string",
  "details": "optional"
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
  'Simpan bearer token toko di backend server merchant, bukan di frontend browser publik.',
  'Pastikan callback_url merchant public, HTTPS, dan merespons cepat dengan HTTP 200.',
  'Saat menerima callback, lakukan idempotency berdasarkan trx_id agar callback ulang tidak memproses order dua kali.',
  'Gunakan custom_ref untuk menghubungkan trx_id gateway dengan order internal merchant.',
  'Anggap status final hanya success, failed, atau expired. Pending yang lewat 20 menit akan di-expire oleh scheduler project.',
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
      description="Panduan lengkap untuk merchant website yang akan terhubung ke project ini, termasuk alur QRIS, callback, keamanan, dan transparansi pergerakan saldo."
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="copyText(apiBaseURL, 'Base URL API')">
          <Copy class="mr-2 h-4 w-4" />
          Copy Base URL
        </Button>
        <Button size="sm" @click="copyText(authHeaderSnippet, 'Header Authorization')">
          <ShieldCheck class="mr-2 h-4 w-4" />
          Copy Auth Header
        </Button>
      </template>
    </PageHeader>

    <Alert class="app-panel">
      <BookOpenText class="h-4 w-4" />
      <AlertTitle>Docs v1.0</AlertTitle>
      <AlertDescription>
        <span class="font-medium text-foreground">Semua request method harus POST</span>
      </AlertDescription>
    </Alert>

    <div class="docs-step-grid">
      <Card v-for="step in quickStartSteps" :key="step.title" class="docs-step-card">
        <CardHeader class="space-y-2">
          <CardTitle class="text-base">{{ step.title }}</CardTitle>
          <CardDescription>{{ step.description }}</CardDescription>
        </CardHeader>
      </Card>
    </div>

    <div class="grid items-start gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(0,0.95fr)]">
      <Card class="app-panel docs-table-card">
        <CardHeader>
          <CardTitle>Merchant Endpoint Catalog</CardTitle>
          <CardDescription>
            <strong class="mt-5">Prefix: </strong><code>/api/v1</code>
          </CardDescription>
        </CardHeader>
        <CardContent class="app-table-shell">
          <Table class="app-data-table docs-table-tight">
            <TableHeader>
              <TableRow>
                <TableHead>Method</TableHead>
                <TableHead>Path</TableHead>
                <TableHead>Auth</TableHead>
                <TableHead>Purpose</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="endpoint in merchantEndpoints" :key="`${endpoint.method}:${endpoint.path}`">
                <TableCell>
                  <Badge :variant="endpoint.method === 'GET' ? 'secondary' : 'default'">
                    {{ endpoint.method }}
                  </Badge>
                </TableCell>
                <TableCell><code>{{ endpoint.path }}</code></TableCell>
                <TableCell>{{ endpoint.auth }}</TableCell>
                <TableCell class="whitespace-normal">{{ endpoint.purpose }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>

          <div class="docs-inline-insights mt-4">
            <div v-for="item in endpointInsights" :key="item.title" class="docs-inline-insight">
              <p class="docs-inline-insight-title">{{ item.title }}</p>
              <p class="docs-inline-insight-copy">{{ item.description }}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Merchant Checklist</CardTitle>
          <CardDescription>
            Checklist singkat supaya integrasi merchant stabil, aman, dan mudah direkonsiliasi.
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

    <div class="grid gap-4 xl:grid-cols-2">
      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Authorization Header</CardTitle>
          <CardDescription>
            Gunakan token toko sebagai Bearer token. Token ini didapat dari halaman Toko dan boleh di-rotate oleh owner toko.
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

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Response Error Format</CardTitle>
          <CardDescription>
            Semua error dari backend mengikuti format terstruktur agar mudah dibaca oleh merchant website.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="docs-code-block">
            <pre>{{ errorResponseSnippet }}</pre>
          </div>
          <p class="text-sm text-muted-foreground">
            Gunakan field <code>message</code> untuk ringkasan dan <code>details</code> untuk penjelasan teknis yang lebih spesifik.
          </p>
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
                Request ini membuat transaksi pending di project lalu mengembalikan QR payload dari external API.
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
              <CardTitle>Check Status Example</CardTitle>
              <CardDescription>
                Cocok dipakai jika merchant ingin polling status manual. Jika snapshot final sudah ada, project akan memanfaatkan cache berdasarkan trx_id.
              </CardDescription>
            </div>
            <Button variant="ghost" size="icon" @click="copyText(statusSnippet, 'Snippet Check Status')">
              <Copy class="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div class="docs-code-block">
            <pre>{{ statusSnippet }}</pre>
          </div>
        </CardContent>
      </Card>
    </div>

    <Card class="app-panel">
      <CardHeader>
        <div class="flex items-center gap-2">
          <Cable class="h-5 w-5 text-(--brand)" />
          <div>
            <CardTitle>Callback Payload ke Merchant Website</CardTitle>
            <CardDescription>
              Inilah payload yang akan dikirim project ke <code>callback_url</code> toko setelah webhook external API tervalidasi dan transaksi diproses.
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="docs-code-block">
          <pre>{{ callbackPayloadSnippet }}</pre>
        </div>
        <Separator />
        <div class="grid gap-4 lg:grid-cols-2">
          <div class="docs-list-item">
            <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <p class="text-sm text-muted-foreground">
              Field <code>merchant_id</code> pada callback ke merchant berisi <span class="font-medium text-foreground">token toko</span>, bukan secret external API.
            </p>
          </div>
          <div class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--success)" />
            <p class="text-sm text-muted-foreground">
              Merchant sebaiknya membalas callback dengan <code>HTTP 200</code> dan body <code>{"success": true}</code> supaya readiness check dan operasi monitoring mudah tervalidasi.
            </p>
          </div>
        </div>
      </CardContent>
    </Card>

    <div class="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-center gap-2">
            <WalletCards class="h-5 w-5 text-(--warning)" />
            <div>
              <CardTitle>Financial Rules yang Berlaku</CardTitle>
              <CardDescription>
                Bagian ini menjelaskan secara transparan bagaimana uang bergerak di dalam project ini.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="docs-code-block">
            <pre>{{ financialFormulaSnippet }}</pre>
          </div>
          <div class="docs-metric-grid">
            <Card class="docs-mini-card docs-metric-card docs-metric-card-profit">
              <CardHeader class="docs-metric-card-header">
                <div class="docs-metric-icon">
                  <TrendingUp class="h-4 w-4" />
                </div>
                <div class="docs-metric-heading">
                  <CardTitle class="text-sm">Project Profit</CardTitle>
                  <CardDescription>3% dari tiap deposit success.</CardDescription>
                </div>
              </CardHeader>
              <CardContent class="docs-metric-card-body">
                <div class="docs-metric-card-value">{{ platformFeePercent }}%</div>
                <div class="docs-metric-eyebrow">Deposit fee</div>
                <p class="docs-metric-card-note">
                  Fee dipotong saat deposit success final lalu dicatat ke ledger.
                </p>
              </CardContent>
            </Card>
            <Card class="docs-mini-card docs-metric-card">
              <CardHeader class="docs-metric-card-header">
                <div class="docs-metric-icon">
                  <ArrowDownUp class="h-4 w-4" />
                </div>
                <div class="docs-metric-heading">
                  <CardTitle class="text-sm">Pending Credit</CardTitle>
                  <CardDescription>Masuk ke pending balance toko.</CardDescription>
                </div>
              </CardHeader>
              <CardContent class="docs-metric-card-body">
                <div class="docs-metric-card-value docs-metric-card-value-sm">Auto Credit</div>
                <div class="docs-metric-eyebrow">Webhook verified</div>
                <p class="docs-metric-card-note">
                  Hak toko masuk ke pending setelah amount dipotong platform fee.
                </p>
              </CardContent>
            </Card>
            <Card class="docs-mini-card docs-metric-card">
              <CardHeader class="docs-metric-card-header">
                <div class="docs-metric-icon">
                  <ReceiptText class="h-4 w-4" />
                </div>
                <div class="docs-metric-heading">
                  <CardTitle class="text-sm">Withdraw Cost</CardTitle>
                  <CardDescription>Biaya vendor, bukan profit project.</CardDescription>
                </div>
              </CardHeader>
              <CardContent class="docs-metric-card-body">
                <div class="docs-metric-card-value docs-metric-card-value-sm">External Fee</div>
                <div class="docs-metric-eyebrow">Vendor transfer</div>
                <p class="docs-metric-card-note">
                  Biaya vendor tetap dibebankan ke settle balance saat withdraw berjalan.
                </p>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <div class="flex items-center gap-2">
            <Landmark class="h-5 w-5 text-(--brand-strong)" />
            <div>
              <CardTitle>Operasional Dashboard</CardTitle>
              <CardDescription>
                Route dashboard bukan API merchant, tetapi penting dipahami agar merchant tahu sumber saldo yang terlihat di panel.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-3">
          <div v-for="note in operationalNotes" :key="note.title" class="docs-list-item">
            <CheckCheck class="mt-0.5 h-4 w-4 shrink-0 text-(--brand)" />
            <div>
              <p class="text-sm font-medium text-foreground">{{ note.title }}</p>
              <p class="text-sm text-muted-foreground">{{ note.description }}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </section>
</template>
