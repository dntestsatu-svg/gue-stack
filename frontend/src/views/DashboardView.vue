<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowDownRight, ArrowUpRight, UserPlus, Users } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import AppIcon from '@/components/AppIcon.vue'
import DashboardStatusAreaChart from '@/components/dashboard/DashboardStatusAreaChart.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { useFormatters } from '@/composables/useFormatters'
import { usePolling } from '@/composables/usePolling'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import * as dashboardApi from '@/services/dashboard'
import { getApiErrorMessage } from '@/services/http'
import type { DashboardOverview, UserRole } from '@/services/types'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const overview = ref<DashboardOverview | null>(null)
const errorMessage = ref('')
const loading = ref(false)
const latestSuccessMarker = ref('')
const successToastReady = ref(false)

const { formatCurrency, formatDateShort, formatNumber, formatPercent } = useFormatters()

const actorRole = computed<UserRole>(() => userStore.profile?.role ?? 'user')
const canManageUsers = computed(() => actorRole.value !== 'user')
const canViewProjectProfit = computed(() => Boolean(overview.value?.can_view_project_profit))
const canViewExternalBalance = computed(() => Boolean(overview.value?.can_view_external_balance))

type DashboardMetricCard = {
  title: string
  value: string
  hint: string
  positive: boolean
}

const loadDashboardData = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const nextOverview = await dashboardApi.fetchOverview()
    const latestOrder = nextOverview.latest_success_orders[0]
    const nextMarker = latestOrder
      ? `${latestOrder.reference ?? latestOrder.id}:${latestOrder.created_at}`
      : ''

    if (successToastReady.value && nextMarker !== '' && nextMarker !== latestSuccessMarker.value) {
      toast.success('Pembayaran sukses baru diterima', {
        description: `${latestOrder?.toko_name ?? 'Toko'} • ${latestOrder?.reference ?? 'No reference'} • ${formatCurrency(latestOrder?.amount ?? 0)}`,
      })
    }

    overview.value = nextOverview
    latestSuccessMarker.value = nextMarker
    successToastReady.value = true
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const { runNow } = usePolling(loadDashboardData, 10000)

const metricCards = computed(() => {
  const metrics = overview.value?.metrics
  const external = overview.value?.external_balance
  const cards: DashboardMetricCard[] = [
    {
      title: 'Success Rate',
      value: formatPercent(metrics?.success_rate ?? 0),
      hint: `${metrics?.success_transactions ?? 0} transaksi sukses`,
      positive: true,
    },
  ]

  if (canViewExternalBalance.value) {
    cards.push(
      {
        title: 'Pending Balance (External)',
        value: formatCurrency(external?.pending_balance ?? 0),
        hint: 'Sumber: payment gateway',
        positive: true,
      },
      {
        title: 'Available Balance (External)',
        value: formatCurrency(external?.available_balance ?? 0),
        hint: 'Sumber: payment gateway',
        positive: true,
      },
    )
  } else {
    cards.push(
      {
        title: 'Total Transaksi Sukses',
        value: formatNumber(metrics?.success_transactions ?? 0),
        hint: 'Scoped sesuai relasi toko akun ini',
        positive: true,
      },
      {
        title: 'Total Deposit Sukses',
        value: formatCurrency(metrics?.success_deposit ?? 0),
        hint: 'Akumulasi nominal deposit sukses',
        positive: true,
      },
    )
  }

  if (canViewProjectProfit.value) {
    cards.push({
      title: 'Total Keuntungan Project',
      value: formatCurrency(metrics?.project_profit ?? 0),
      hint: 'Visible for dev role only',
      positive: true,
    })
  } else {
    cards.push({
      title: 'Transaksi Pending',
      value: formatNumber(metrics?.pending_transactions ?? 0),
      hint: 'Masih menunggu penyelesaian status',
      positive: (metrics?.pending_transactions ?? 0) === 0,
    })
  }

  return cards
})
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Control Center"
      title="Enterprise Operations Dashboard"
      description="Realtime monitoring transaksi gateway dan kesehatan operasional."
      :updated-at="overview?.updated_at"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="loading" @click="runNow">
          {{ loading ? 'Refreshing...' : 'Refresh' }}
        </Button>
        <Button v-if="canManageUsers" size="sm" @click="router.push('/users?create=1')">
          <UserPlus class="mr-2 h-4 w-4" />
          Add User
        </Button>
        <Button v-if="canManageUsers" size="sm" variant="outline" @click="router.push('/users')">
          <Users class="mr-2 h-4 w-4" />
          List Users
        </Button>
      </template>
    </PageHeader>

    <p
      v-if="errorMessage"
      class="rounded-md border border-(--danger)/25 bg-(--danger)/8 px-3 py-2 text-sm text-(--danger)"
    >
      {{ errorMessage }}
    </p>
    <p
      v-if="canViewExternalBalance && overview?.external_balance_error"
      class="rounded-md border border-(--warning)/30 bg-(--warning)/10 px-3 py-2 text-sm text-(--warning)"
    >
      External balance warning: {{ overview.external_balance_error }}
    </p>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <template v-if="loading && !overview">
        <Card v-for="idx in 4" :key="idx" class="dashboard-kpi-card">
          <CardHeader class="pb-2">
            <Skeleton class="h-4 w-36" />
            <Skeleton class="h-7 w-24" />
          </CardHeader>
        </Card>
      </template>
      <template v-else>
        <Card
          v-for="item in metricCards"
          :key="item.title"
          class="dashboard-kpi-card"
        >
          <CardHeader class="space-y-2 pb-2">
            <CardDescription>{{ item.title }}</CardDescription>
            <CardTitle class="text-2xl">{{ item.value }}</CardTitle>
            <div class="text-muted-foreground flex items-center gap-1 text-xs">
              <AppIcon :icon="item.positive ? ArrowUpRight : ArrowDownRight" class="h-3.5 w-3.5" />
              {{ item.hint }}
            </div>
          </CardHeader>
        </Card>
      </template>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1.5fr_1fr]">
      <Card class="app-panel">
        <CardHeader class="pb-1">
          <CardTitle>Success vs Failed / Expired</CardTitle>
          <CardDescription>Rolling 12 jam terakhir berdasarkan status transaksi.</CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="loading && !overview" class="space-y-2">
            <Skeleton class="h-64 w-full" />
          </div>
          <DashboardStatusAreaChart
            v-else-if="(overview?.status_series?.length ?? 0) > 0"
            :series="overview?.status_series ?? []"
          />
          <EmptyState
            v-else
            title="Belum Ada Data Chart"
            description="Data chart akan tampil setelah ada transaksi dalam jendela waktu aktif."
          />
          <div class="mt-3 flex flex-wrap gap-3 text-sm text-muted-foreground">
            <span>Total: {{ overview?.metrics.total_transactions ?? 0 }}</span>
            <span>Success: {{ overview?.metrics.success_transactions ?? 0 }}</span>
            <span>Pending: {{ overview?.metrics.pending_transactions ?? 0 }}</span>
            <span>Failed: {{ overview?.metrics.failed_transactions ?? 0 }}</span>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader class="pb-1">
          <CardTitle>Flow Summary</CardTitle>
          <CardDescription>Ringkasan nominal transaksi sukses.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3 text-sm">
          <div class="dashboard-stat-row">
            <span>Success Deposit</span>
            <strong>{{ formatCurrency(overview?.metrics.success_deposit ?? 0) }}</strong>
          </div>
          <div class="dashboard-stat-row">
            <span>Success Withdraw</span>
            <strong>{{ formatCurrency(overview?.metrics.success_withdraw ?? 0) }}</strong>
          </div>
          <div class="dashboard-stat-row">
            <span>Net Flow</span>
            <strong :class="(overview?.metrics.net_flow ?? 0) >= 0 ? 'text-(--success)' : 'text-(--danger)'">
              {{ formatCurrency(Math.abs(overview?.metrics.net_flow ?? 0)) }}
            </strong>
          </div>
        </CardContent>
      </Card>
    </div>

    <Card class="app-panel">
      <CardHeader>
        <CardTitle>Latest Order (Success)</CardTitle>
        <CardDescription>Order sukses terbaru dari semua toko milik user saat ini.</CardDescription>
      </CardHeader>
      <CardContent>
        <div v-if="loading && !overview" class="space-y-2">
          <Skeleton class="h-12 w-full" />
          <Skeleton class="h-12 w-full" />
        </div>
        <EmptyState
          v-else-if="(overview?.latest_success_orders?.length ?? 0) === 0"
          title="Belum Ada Order Sukses"
          description="Transaksi sukses pertama akan muncul di tabel ini."
          action-label="Ke Histori Transaksi"
          @action="router.push('/histori-transaksi')"
        />
        <div v-else class="overflow-hidden rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waktu</TableHead>
                <TableHead>Toko</TableHead>
                <TableHead>Reference</TableHead>
                <TableHead class="text-right">Amount</TableHead>
                <TableHead class="text-right">Netto</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="item in overview?.latest_success_orders" :key="item.id">
                <TableCell>{{ formatDateShort(item.created_at) }}</TableCell>
                <TableCell class="font-medium">{{ item.toko_name }}</TableCell>
                <TableCell>{{ item.reference || '-' }}</TableCell>
                <TableCell class="text-right">{{ formatCurrency(item.amount) }}</TableCell>
                <TableCell class="text-right">{{ formatCurrency(item.netto) }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  </section>
</template>
