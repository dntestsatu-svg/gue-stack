<template>
  <section class="space-y-6">
    <header class="dashboard-hero">
      <div class="space-y-2">
        <p class="dashboard-eyebrow">Toko Workspace</p>
        <h1 class="text-2xl font-semibold tracking-tight md:text-3xl">Manage Toko & Settlement</h1>
        <p class="text-sm text-[var(--muted-foreground)]">
          Available balance & settlement balance dikelola dari settlement internal.
          <span v-if="lastUpdated" class="ml-1">Updated {{ formatTime(lastUpdated) }}</span>
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" @click="runNow">Refresh</Button>
        <Button size="sm" variant="ghost" @click="scrollToManageSection">Manage Toko</Button>
        <Button v-if="canCreateTokoRole" size="sm" @click="openCreateTokoModal">Create Toko</Button>
      </div>
    </header>

    <p v-if="errorMessage" class="rounded-md border border-[var(--danger)]/25 bg-[var(--danger)]/8 px-3 py-2 text-sm text-[var(--danger)]">
      {{ errorMessage }}
    </p>
    <p v-if="createErrorMessage" class="rounded-md border border-[var(--danger)]/25 bg-[var(--danger)]/8 px-3 py-2 text-sm text-[var(--danger)]">
      {{ createErrorMessage }}
    </p>

    <div class="grid gap-4 md:grid-cols-3">
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Total Settlement Balance</CardDescription>
          <CardTitle class="text-2xl">{{ formatCurrencyWithDecimals(totalSettlementBalance) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Total Available Balance</CardDescription>
          <CardTitle class="text-2xl">{{ formatCurrencyWithDecimals(totalAvailableBalance) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Visible Toko</CardDescription>
          <CardTitle class="text-2xl">{{ tokos.length }}</CardTitle>
        </CardHeader>
      </Card>
    </div>

    <section ref="manageSectionRef">
      <Card class="app-panel border-none">
        <CardHeader>
          <CardTitle>Toko Management</CardTitle>
          <CardDescription>
            Token digunakan sebagai Bearer pada internal payment endpoint.
          </CardDescription>
        </CardHeader>
        <CardContent class="overflow-x-auto">
          <table class="w-full min-w-[920px] text-sm">
            <thead>
              <tr class="border-b border-[var(--border)] text-left text-[var(--muted-foreground)]">
                <th class="px-3 py-3 font-medium">Toko</th>
                <th class="px-3 py-3 font-medium">Charge</th>
                <th class="px-3 py-3 font-medium">Callback URL</th>
                <th class="px-3 py-3 font-medium">Token</th>
                <th class="px-3 py-3 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in tokos"
                :key="item.id"
                class="border-b border-[var(--border)]/70 transition hover:bg-[var(--background-muted)]/40"
              >
                <td class="px-3 py-3 font-medium">{{ item.name }}</td>
                <td class="px-3 py-3">{{ item.charge }}%</td>
                <td class="px-3 py-3">{{ item.callback_url || '-' }}</td>
                <td class="px-3 py-3 font-mono text-xs">{{ item.token }}</td>
                <td class="px-3 py-3">
                  <Button size="sm" variant="outline" type="button" @click="copyToClipboard(item.token)">Copy Token</Button>
                </td>
              </tr>
              <tr v-if="!loading && tokos.length === 0">
                <td colspan="5" class="px-3 py-8 text-center text-[var(--muted-foreground)]">Belum ada toko.</td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>
    </section>

    <Card class="app-panel border-none">
      <CardHeader>
        <CardTitle>Settlement Balances</CardTitle>
        <CardDescription>Manual settlement hanya untuk role dev. Role lain tetap melihat data secara read-only.</CardDescription>
      </CardHeader>
      <CardContent class="overflow-x-auto">
        <table class="w-full min-w-[980px] text-sm">
          <thead>
            <tr class="border-b border-[var(--border)] text-left text-[var(--muted-foreground)]">
              <th class="px-3 py-3 font-medium">Toko</th>
              <th class="px-3 py-3 font-medium text-right">Settlement Balance</th>
              <th class="px-3 py-3 font-medium text-right">Available Balance</th>
              <th class="px-3 py-3 font-medium">Updated</th>
              <th class="px-3 py-3 font-medium">Settlement Action</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in balances"
              :key="item.toko_id"
              class="border-b border-[var(--border)]/60"
            >
              <td class="px-3 py-3 font-medium">{{ item.toko_name }}</td>
              <td class="px-3 py-3 text-right">{{ formatCurrencyWithDecimals(item.settlement_balance) }}</td>
              <td class="px-3 py-3 text-right">{{ formatCurrencyWithDecimals(item.available_balance) }}</td>
              <td class="px-3 py-3">{{ formatDate(item.updated_at) }}</td>
              <td class="px-3 py-3">
                <form v-if="canManualSettlement" class="grid gap-2 md:grid-cols-[1fr_auto]" @submit.prevent="submitSettlement(item.toko_id)">
                  <Input
                    v-model="formByToko[item.toko_id].settlementBalance"
                    type="number"
                    step="0.01"
                    min="0"
                    placeholder="Settlement amount"
                  />
                  <Button
                    type="submit"
                    size="sm"
                    :disabled="formByToko[item.toko_id].loading"
                  >
                    {{ formByToko[item.toko_id].loading ? 'Saving...' : 'Apply' }}
                  </Button>
                </form>
                <span v-else class="text-sm text-[var(--muted-foreground)]">Developer only</span>
              </td>
            </tr>
            <tr v-if="!loading && balances.length === 0">
              <td colspan="5" class="px-3 py-8 text-center text-[var(--muted-foreground)]">
                Belum ada data balance toko.
              </td>
            </tr>
          </tbody>
        </table>
      </CardContent>
    </Card>
  </section>

  <Teleport to="body">
    <div v-if="showCreateTokoModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button class="absolute inset-0 bg-black/55" type="button" aria-label="Close modal" @click="closeCreateTokoModal" />
      <Card class="relative z-10 w-full max-w-lg border border-[var(--border)] shadow-2xl">
        <CardHeader>
          <CardTitle>Create Toko</CardTitle>
          <CardDescription>Maksimal 3 toko per creator divalidasi backend. Token dibuat otomatis oleh sistem.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <form class="space-y-4" @submit.prevent="submitCreateToko">
            <div class="space-y-2">
              <Label for="toko-name">Nama Toko</Label>
              <Input id="toko-name" v-model="createForm.name" placeholder="Contoh: Toko Alfa" />
            </div>
            <div class="space-y-2">
              <Label for="callback-url">Callback URL (opsional)</Label>
              <Input id="callback-url" v-model="createForm.callbackURL" placeholder="https://domain/callback" />
            </div>
            <p class="text-sm text-[var(--muted-foreground)]">
              Available balance akan dihitung ulang otomatis ketika settlement dijalankan oleh developer.
            </p>
            <div class="flex items-center justify-end gap-2">
              <Button type="button" variant="ghost" :disabled="createLoading" @click="closeCreateTokoModal">Cancel</Button>
              <Button type="submit" :disabled="createLoading || !canCreateTokoRole">
                {{ createLoading ? 'Creating...' : 'Create Toko' }}
              </Button>
            </div>
          </form>

          <div v-if="createdToko" class="rounded-lg border border-[var(--success)]/35 bg-[var(--success)]/10 p-3">
            <p class="text-sm font-semibold text-[var(--success)]">Toko berhasil dibuat: {{ createdToko.name }}</p>
            <p class="mt-1 text-xs text-[var(--muted-foreground)]">Simpan token ini untuk header Bearer.</p>
            <div class="mt-2 flex items-center gap-2">
              <code class="min-w-0 flex-1 truncate rounded bg-[var(--background-muted)] px-2 py-1 text-xs">{{ createdToko.token }}</code>
              <Button size="sm" variant="outline" type="button" @click="copyToClipboard(createdToko.token)">Copy</Button>
            </div>
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
import * as tokoApi from '@/services/toko'
import type { TokoBalanceItem, TokoItem } from '@/services/types'
import { useUserStore } from '@/stores/user'

type SettlementFormState = {
  settlementBalance: string
  loading: boolean
}

const loading = ref(false)
const balances = ref<TokoBalanceItem[]>([])
const tokos = ref<TokoItem[]>([])
const errorMessage = ref('')
const lastUpdated = ref('')
const formByToko = reactive<Record<number, SettlementFormState>>({})
const userStore = useUserStore()
const { formatCurrency, formatDateMedium, formatTime } = useFormatters()
const manageSectionRef = ref<unknown>(null)

const showCreateTokoModal = ref(false)
const createLoading = ref(false)
const createErrorMessage = ref('')
const createdToko = ref<TokoItem | null>(null)
const createForm = reactive({
  name: '',
  callbackURL: '',
})

const canManualSettlement = computed(() => {
  const role = userStore.profile?.role
  return role === 'dev'
})

const canCreateTokoRole = computed(() => {
  const role = userStore.profile?.role
  return role === 'dev' || role === 'superadmin' || role === 'admin'
})

const totalSettlementBalance = computed(() =>
  balances.value.reduce((acc, item) => acc + item.settlement_balance, 0),
)
const totalAvailableBalance = computed(() =>
  balances.value.reduce((acc, item) => acc + item.available_balance, 0),
)

const ensureFormState = (item: TokoBalanceItem) => {
  if (!formByToko[item.toko_id]) {
    formByToko[item.toko_id] = {
      settlementBalance: '',
      loading: false,
    }
  }
}

const syncFormWithBalances = () => {
  for (const item of balances.value) {
    ensureFormState(item)
  }
}

const loadTokoData = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const [balanceItems, tokoItems] = await Promise.all([
      tokoApi.fetchBalances(),
      tokoApi.fetchTokos(),
    ])
    balances.value = balanceItems
    tokos.value = tokoItems
    syncFormWithBalances()
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const { runNow } = usePolling(loadTokoData, 10000)

const scrollToManageSection = () => {
  const element = manageSectionRef.value as { scrollIntoView?: (options?: unknown) => void } | null
  element?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
}

const submitSettlement = async (tokoID: number) => {
  const form = formByToko[tokoID]
  if (!form) {
    return
  }

  errorMessage.value = ''
  const settlementBalance = Number(form.settlementBalance)
  if (!Number.isFinite(settlementBalance) || settlementBalance <= 0) {
    errorMessage.value = 'Masukkan nominal settlement yang valid.'
    return
  }

  form.loading = true
  try {
    const updated = await tokoApi.applySettlement(tokoID, {
      settlement_balance: settlementBalance,
    })
    balances.value = balances.value.map((item) =>
      item.toko_id === tokoID ? updated : item,
    )
    form.settlementBalance = ''
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    form.loading = false
  }
}

const openCreateTokoModal = () => {
  if (!canCreateTokoRole.value) {
    createErrorMessage.value = 'Role user tidak memiliki izin membuat toko.'
    return
  }
  createdToko.value = null
  createErrorMessage.value = ''
  createForm.name = ''
  createForm.callbackURL = ''
  showCreateTokoModal.value = true
}

const closeCreateTokoModal = () => {
  if (createLoading.value) {
    return
  }
  showCreateTokoModal.value = false
}

const submitCreateToko = async () => {
  if (!canCreateTokoRole.value) {
    createErrorMessage.value = 'Role user tidak memiliki izin membuat toko.'
    return
  }

  createErrorMessage.value = ''
  createLoading.value = true
  try {
    const payload = {
      name: createForm.name.trim(),
      callback_url: createForm.callbackURL.trim() || undefined,
    }
    const created = await tokoApi.createToko(payload)
    createdToko.value = created
    createForm.name = ''
    createForm.callbackURL = ''
    await runNow()
  } catch (error) {
    createErrorMessage.value = getApiErrorMessage(error)
  } finally {
    createLoading.value = false
  }
}

const copyToClipboard = async (value: string) => {
  if (typeof window === 'undefined' || !window.navigator?.clipboard) {
    return
  }
  try {
    await window.navigator.clipboard.writeText(value)
  } catch {
    // Ignore clipboard failures silently.
  }
}

const formatDate = (value: string) => formatDateMedium(value)
const formatCurrencyWithDecimals = (value: number) => formatCurrency(value, 2)
</script>
