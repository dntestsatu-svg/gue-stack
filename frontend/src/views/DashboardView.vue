<template>
  <section class="space-y-6">
    <header class="dashboard-hero">
      <div class="space-y-2">
        <p class="dashboard-eyebrow">Control Center</p>
        <h1 class="text-2xl font-semibold tracking-tight md:text-3xl">Enterprise Operations Dashboard</h1>
        <p class="text-sm text-[var(--muted-foreground)]">
          Realtime monitoring transaksi gateway dan kesehatan operasional.
          <span v-if="overview?.updated_at" class="ml-1">Updated {{ formatTime(overview.updated_at) }}</span>
        </p>
      </div>
      <div v-if="canManageUsers" class="flex flex-wrap items-center gap-2">
        <Button size="sm" @click="openAddUserModal">Add User</Button>
        <Button size="sm" variant="outline" @click="openUsersModal">List Users</Button>
      </div>
    </header>

    <p v-if="errorMessage" class="rounded-md border border-[var(--danger)]/25 bg-[var(--danger)]/8 px-3 py-2 text-sm text-[var(--danger)]">
      {{ errorMessage }}
    </p>
    <p
      v-if="overview?.external_balance_error"
      class="rounded-md border border-[var(--warning)]/30 bg-[var(--warning)]/10 px-3 py-2 text-sm text-[var(--warning)]"
    >
      External balance warning: {{ overview.external_balance_error }}
    </p>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <Card v-if="canViewProjectProfit" class="dashboard-kpi-card dashboard-kpi-card-profit">
        <CardHeader class="pb-2">
          <CardDescription>Total Keuntungan Project</CardDescription>
          <CardTitle class="text-2xl text-[var(--success)]">{{ formatCurrency(overview?.metrics.project_profit ?? 0) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Pending Balance (External)</CardDescription>
          <CardTitle class="text-2xl">{{ formatCurrency(overview?.external_balance.pending_balance ?? 0) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Available Balance (External)</CardDescription>
          <CardTitle class="text-2xl">{{ formatCurrency(overview?.external_balance.available_balance ?? 0) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Success Rate</CardDescription>
          <CardTitle class="text-2xl">{{ formatPercent(overview?.metrics.success_rate ?? 0) }}</CardTitle>
        </CardHeader>
      </Card>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1.5fr_1fr]">
      <Card class="app-panel border-none">
        <CardHeader class="pb-1">
          <CardTitle>Success vs Failed / Expired</CardTitle>
          <CardDescription>Rolling 12 jam terakhir berdasarkan status transaksi.</CardDescription>
        </CardHeader>
        <CardContent>
          <div class="relative h-64 w-full rounded-lg border border-[var(--border)] bg-[var(--background-muted)]/35 p-3">
            <svg viewBox="0 0 100 40" class="h-full w-full" preserveAspectRatio="none">
              <g stroke="var(--chart-grid)" stroke-width="0.25">
                <line v-for="line in 5" :key="line" x1="0" :y1="line * 8" x2="100" :y2="line * 8" />
              </g>
              <polyline
                fill="none"
                stroke="var(--chart-success)"
                stroke-width="1.2"
                stroke-linecap="round"
                stroke-linejoin="round"
                :points="successPolyline"
              />
              <polyline
                fill="none"
                stroke="var(--chart-failed)"
                stroke-width="1.2"
                stroke-linecap="round"
                stroke-linejoin="round"
                :points="failedPolyline"
              />
            </svg>
          </div>
          <div class="mt-3 flex flex-wrap gap-3 text-sm text-[var(--muted-foreground)]">
            <span>Total: {{ overview?.metrics.total_transactions ?? 0 }}</span>
            <span>Success: {{ overview?.metrics.success_transactions ?? 0 }}</span>
            <span>Pending: {{ overview?.metrics.pending_transactions ?? 0 }}</span>
            <span>Failed: {{ overview?.metrics.failed_transactions ?? 0 }}</span>
          </div>
        </CardContent>
      </Card>

      <Card class="app-panel border-none">
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
            <strong :class="(overview?.metrics.net_flow ?? 0) >= 0 ? 'text-[var(--success)]' : 'text-[var(--danger)]'">
              {{ formatCurrency(Math.abs(overview?.metrics.net_flow ?? 0)) }}
            </strong>
          </div>
        </CardContent>
      </Card>
    </div>

    <Card class="app-panel border-none">
      <CardHeader>
        <CardTitle>Latest Order (Success)</CardTitle>
        <CardDescription>Order sukses terbaru dari semua toko milik user saat ini.</CardDescription>
      </CardHeader>
      <CardContent class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-sm">
          <thead>
            <tr class="border-b border-[var(--border)] text-left text-[var(--muted-foreground)]">
              <th class="px-2 py-2 font-medium">Waktu</th>
              <th class="px-2 py-2 font-medium">Toko</th>
              <th class="px-2 py-2 font-medium">Reference</th>
              <th class="px-2 py-2 font-medium text-right">Amount</th>
              <th class="px-2 py-2 font-medium text-right">Netto</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in overview?.latest_success_orders ?? []"
              :key="item.id"
              class="border-b border-[var(--border)]/60"
            >
              <td class="px-2 py-2">{{ formatDate(item.created_at) }}</td>
              <td class="px-2 py-2">{{ item.toko_name }}</td>
              <td class="px-2 py-2">{{ item.reference || '-' }}</td>
              <td class="px-2 py-2 text-right">{{ formatCurrency(item.amount) }}</td>
              <td class="px-2 py-2 text-right">{{ formatCurrency(item.netto) }}</td>
            </tr>
            <tr v-if="(overview?.latest_success_orders?.length ?? 0) === 0">
              <td colspan="5" class="px-2 py-8 text-center text-[var(--muted-foreground)]">Belum ada order sukses.</td>
            </tr>
          </tbody>
        </table>
      </CardContent>
    </Card>
  </section>

  <Teleport to="body">
    <div v-if="showAddUserModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button class="absolute inset-0 bg-black/55" type="button" aria-label="Close modal" @click="closeAddUserModal" />
      <Card class="relative z-10 w-full max-w-lg border border-[var(--border)] shadow-2xl">
        <CardHeader>
          <CardTitle>Add User</CardTitle>
          <CardDescription>Create akun baru untuk tim internal.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <form class="space-y-4" @submit.prevent="submitAddUser">
            <div class="grid gap-2 md:grid-cols-2">
              <div class="space-y-2">
                <Label for="user-name">Name</Label>
                <Input id="user-name" v-model="addUserForm.name" placeholder="Nama lengkap" />
              </div>
              <div class="space-y-2">
                <Label for="user-email">Email</Label>
                <Input id="user-email" v-model="addUserForm.email" type="email" placeholder="email@domain.com" />
              </div>
            </div>
            <div class="grid gap-2 md:grid-cols-2">
              <div class="space-y-2">
                <Label for="user-password">Password</Label>
                <Input id="user-password" v-model="addUserForm.password" type="password" placeholder="Minimal 8 karakter" />
              </div>
              <div class="space-y-2">
                <Label for="user-role">Role</Label>
                <select
                  id="user-role"
                  v-model="addUserForm.role"
                  class="h-10 w-full rounded-md border border-[var(--border)] bg-[var(--background-elevated)] px-3 text-sm"
                >
                  <option v-for="role in roleOptions" :key="role" :value="role">{{ role }}</option>
                </select>
              </div>
            </div>
            <label class="inline-flex items-center gap-2 text-sm text-[var(--muted-foreground)]">
              <input v-model="addUserForm.isActive" type="checkbox" class="h-4 w-4 rounded border-[var(--border)]" />
              Aktifkan user setelah dibuat
            </label>
            <p v-if="addUserErrorMessage" class="text-sm text-[var(--danger)]">{{ addUserErrorMessage }}</p>
            <div class="flex items-center justify-end gap-2">
              <Button type="button" variant="ghost" :disabled="addUserLoading" @click="closeAddUserModal">Cancel</Button>
              <Button type="submit" :disabled="addUserLoading">
                {{ addUserLoading ? 'Saving...' : 'Create User' }}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  </Teleport>

  <Teleport to="body">
    <div v-if="showUsersModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button class="absolute inset-0 bg-black/55" type="button" aria-label="Close modal" @click="closeUsersModal" />
      <Card class="relative z-10 w-full max-w-4xl border border-[var(--border)] shadow-2xl">
        <CardHeader class="flex flex-row items-center justify-between gap-3">
          <div>
            <CardTitle>User Directory</CardTitle>
            <CardDescription>Daftar akun internal dengan role dan status aktif.</CardDescription>
          </div>
          <Button size="sm" variant="outline" :disabled="usersLoading" @click="loadUsers">
            {{ usersLoading ? 'Loading...' : 'Refresh' }}
          </Button>
        </CardHeader>
        <CardContent>
          <p v-if="usersErrorMessage" class="mb-3 text-sm text-[var(--danger)]">{{ usersErrorMessage }}</p>
          <div class="overflow-x-auto">
            <table class="w-full min-w-[680px] text-sm">
              <thead>
                <tr class="border-b border-[var(--border)] text-left text-[var(--muted-foreground)]">
                  <th class="px-2 py-2 font-medium">Name</th>
                  <th class="px-2 py-2 font-medium">Email</th>
                  <th class="px-2 py-2 font-medium">Role</th>
                  <th class="px-2 py-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in users" :key="item.id" class="border-b border-[var(--border)]/60">
                  <td class="px-2 py-2">{{ item.name }}</td>
                  <td class="px-2 py-2">{{ item.email }}</td>
                  <td class="px-2 py-2">
                    <span class="status-pill bg-[var(--background-muted)] text-[var(--foreground)]">{{ item.role }}</span>
                  </td>
                  <td class="px-2 py-2">
                    <span
                      class="status-pill"
                      :class="item.is_active ? 'bg-[color-mix(in_oklab,var(--success)_20%,transparent)] text-[var(--success)]' : 'bg-[color-mix(in_oklab,var(--danger)_20%,transparent)] text-[var(--danger)]'"
                    >
                      {{ item.is_active ? 'active' : 'inactive' }}
                    </span>
                  </td>
                </tr>
                <tr v-if="!usersLoading && users.length === 0">
                  <td colspan="4" class="px-2 py-8 text-center text-[var(--muted-foreground)]">Belum ada user.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useFormatters } from '@/composables/useFormatters'
import { usePolling } from '@/composables/usePolling'
import { getApiErrorMessage } from '@/services/http'
import * as dashboardApi from '@/services/dashboard'
import * as userApi from '@/services/user'
import type { DashboardOverview, User, UserRole } from '@/services/types'
import { useUserStore } from '@/stores/user'

const overview = ref<DashboardOverview | null>(null)
const errorMessage = ref('')
const userStore = useUserStore()

const showAddUserModal = ref(false)
const addUserLoading = ref(false)
const addUserErrorMessage = ref('')
const addUserForm = reactive({
  name: '',
  email: '',
  password: '',
  role: 'user' as UserRole,
  isActive: true,
})

const showUsersModal = ref(false)
const usersLoading = ref(false)
const usersErrorMessage = ref('')
const users = ref<User[]>([])

const { formatCurrency, formatDateShort, formatPercent, formatTime } = useFormatters()

const actorRole = computed(() => userStore.profile?.role ?? 'user')
const canManageUsers = computed(() => actorRole.value !== 'user')
const canViewProjectProfit = computed(() => actorRole.value === 'dev' && Boolean(overview.value?.can_view_project_profit))
const roleOptions = computed<UserRole[]>(() => {
  if (actorRole.value === 'dev') {
    return ['superadmin', 'admin', 'user']
  }
  if (actorRole.value === 'superadmin') {
    return ['admin', 'user']
  }
  if (actorRole.value === 'admin') {
    return ['user']
  }
  return []
})

const loadDashboardData = async () => {
  errorMessage.value = ''
  try {
    overview.value = await dashboardApi.fetchOverview()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  }
}

const { runNow } = usePolling(loadDashboardData, 10000)

const maxSeriesValue = computed(() => {
  const series = overview.value?.status_series ?? []
  let maxValue = 0
  for (const item of series) {
    maxValue = Math.max(maxValue, item.success_count, item.failed_expired_count)
  }
  return maxValue || 1
})

const toPolyline = (selector: 'success_count' | 'failed_expired_count') => {
  const series = overview.value?.status_series ?? []
  if (series.length === 0) {
    return ''
  }
  return series
    .map((item, idx) => {
      const x = series.length === 1 ? 50 : (idx / (series.length - 1)) * 100
      const normalized = item[selector] / maxSeriesValue.value
      const y = 36 - normalized * 30
      return `${x},${y}`
    })
    .join(' ')
}

const successPolyline = computed(() => toPolyline('success_count'))
const failedPolyline = computed(() => toPolyline('failed_expired_count'))

const resetAddUserForm = () => {
  addUserForm.name = ''
  addUserForm.email = ''
  addUserForm.password = ''
  addUserForm.role = roleOptions.value[0] ?? 'user'
  addUserForm.isActive = true
}

const openAddUserModal = () => {
  if (!canManageUsers.value) {
    return
  }
  addUserErrorMessage.value = ''
  resetAddUserForm()
  showAddUserModal.value = true
}

const closeAddUserModal = () => {
  if (addUserLoading.value) {
    return
  }
  showAddUserModal.value = false
}

const submitAddUser = async () => {
  addUserErrorMessage.value = ''
  addUserLoading.value = true
  try {
    await userApi.create({
      name: addUserForm.name.trim(),
      email: addUserForm.email.trim(),
      password: addUserForm.password,
      role: addUserForm.role,
      is_active: addUserForm.isActive,
    })
    showAddUserModal.value = false
    if (showUsersModal.value) {
      await loadUsers()
    }
    await runNow()
  } catch (error) {
    addUserErrorMessage.value = getApiErrorMessage(error)
  } finally {
    addUserLoading.value = false
  }
}

const loadUsers = async () => {
  usersLoading.value = true
  usersErrorMessage.value = ''
  try {
    users.value = await userApi.list(100)
  } catch (error) {
    usersErrorMessage.value = getApiErrorMessage(error)
  } finally {
    usersLoading.value = false
  }
}

const openUsersModal = async () => {
  if (!canManageUsers.value) {
    return
  }
  showUsersModal.value = true
  await loadUsers()
}

const closeUsersModal = () => {
  if (usersLoading.value) {
    return
  }
  showUsersModal.value = false
}

const formatDate = (value: string) => formatDateShort(value)
</script>
