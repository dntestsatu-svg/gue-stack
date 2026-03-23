<script setup lang="ts">
import { computed } from 'vue'
import {
  BookOpenText,
  Cable,
  CheckCheck,
  Copy,
  Landmark,
  ShieldCheck,
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

const platformFeePercent = 3
const apiBaseURL = computed(() => `${resolveApiBaseURL(import.meta.env.VITE_API_BASE_URL)}/api/v1`)

const quickStartSteps: StepDoc[] = [
  {
    title: '1. Ambil Token Toko',
    description: 'Token toko didapat dari route Toko. Token ini dipakai sebagai Bearer token saat merchant website memanggil API project ini.',
  },
  {
    title: '2. Simpan Callback URL',
    description: 'Set callback_url toko ke endpoint server merchant yang akan menerima update transaksi final dari project ini.',
  },
  {
    title: '3. Generate QRIS',
    description: 'Merchant website memanggil endpoint generate. Project ini akan meneruskan request ke external API, menyimpan transaksi pending, lalu mengembalikan QR payload.',
  },
  {
    title: '4. Terima Callback Final',
    description: 'Setelah external API mengirim webhook success/failed/expired ke project ini, project akan memprosesnya secara idempotent lalu callback ke merchant website.',
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
    title: 'Withdraw Fee',
    description: 'Fee withdraw external tidak dihitung sebagai profit project. Fee ini tetap membebani saldo settle toko pada saat withdraw.',
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
      <AlertTitle>Transparansi Arsitektur</AlertTitle>
      <AlertDescription>
        Merchant flow pada project ini adalah
        <span class="font-medium text-foreground">merchant website -> this project -> external API -> this project -> merchant website</span>.
        Merchant hanya memakai token toko miliknya sendiri. Secret dan kredensial external API tidak pernah diekspos ke merchant.
      </AlertDescription>
    </Alert>

    <div class="grid gap-4 xl:grid-cols-4">
      <Card v-for="step in quickStartSteps" :key="step.title" class="dashboard-kpi-card">
        <CardHeader class="space-y-2">
          <CardTitle class="text-base">{{ step.title }}</CardTitle>
          <CardDescription>{{ step.description }}</CardDescription>
        </CardHeader>
      </Card>
    </div>

    <div class="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(0,0.95fr)]">
      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Merchant Endpoint Catalog</CardTitle>
          <CardDescription>
            Endpoint di bawah ini adalah endpoint yang umumnya dipakai merchant website. Semua request memakai bearer token toko.
          </CardDescription>
        </CardHeader>
        <CardContent class="app-table-shell">
          <Table class="app-data-table">
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
                <TableCell class="min-w-80 whitespace-normal">{{ endpoint.purpose }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
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
          <div class="grid gap-3 md:grid-cols-3">
            <Card class="docs-mini-card">
              <CardHeader class="space-y-1 pb-3">
                <CardTitle class="text-sm">Project Profit</CardTitle>
                <CardDescription>Hanya dari platform fee deposit sukses.</CardDescription>
              </CardHeader>
              <CardContent class="pt-0">
                <Badge>{{ platformFeePercent }}% per success deposit</Badge>
              </CardContent>
            </Card>
            <Card class="docs-mini-card">
              <CardHeader class="space-y-1 pb-3">
                <CardTitle class="text-sm">Pending Balance Toko</CardTitle>
                <CardDescription>Naik otomatis setelah deposit sukses difinalisasi.</CardDescription>
              </CardHeader>
            </Card>
            <Card class="docs-mini-card">
              <CardHeader class="space-y-1 pb-3">
                <CardTitle class="text-sm">Withdraw Fee</CardTitle>
                <CardDescription>Bukan profit project dan tetap membebani settle balance toko.</CardDescription>
              </CardHeader>
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
