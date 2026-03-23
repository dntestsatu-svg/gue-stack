<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Copy, KeyRound, PencilLine, Plus, Search, TriangleAlert } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { usePolling } from '@/composables/usePolling'
import { getApiErrorMessage } from '@/services/http'
import * as tokoApi from '@/services/toko'
import type { TokoItem, TokoWorkspaceItem, TokoWorkspacePage } from '@/services/types'
import { useUserStore } from '@/stores/user'

type SettlementFormState = {
  settlementBalance: string
  loading: boolean
}

const userStore = useUserStore()
const { formatCurrency, formatDateMedium } = useFormatters()

const workspace = ref<TokoWorkspacePage | null>(null)
const loading = ref(false)
const errorMessage = ref('')
const lastUpdated = ref('')
const createDialogOpen = ref(false)
const createLoading = ref(false)
const createErrorMessage = ref('')
const createdToko = ref<TokoItem | null>(null)
const manageDialogOpen = ref(false)
const manageLoading = ref(false)
const manageErrorMessage = ref('')
const regenerateTokenLoading = ref(false)
const managedToko = ref<TokoWorkspaceItem | null>(null)
const managedToken = ref('')

const filters = reactive({
  q: '',
})

const pagination = reactive({
  limit: 10,
  offset: 0,
  total: 0,
  hasMore: false,
})

const formByToko = reactive<Record<number, SettlementFormState>>({})
const createForm = reactive({
  name: '',
  callbackURL: '',
})
const manageForm = reactive({
  name: '',
  callbackURL: '',
})

const canManualSettlement = computed(() => userStore.profile?.role === 'dev')
const canCreateTokoRole = computed(() => {
  const role = userStore.profile?.role
  return role === 'dev' || role === 'superadmin' || role === 'admin'
})

const items = computed(() => workspace.value?.items ?? [])
const summary = computed(() => workspace.value?.summary ?? {
  total_tokos: 0,
  total_pending_balance: 0,
  total_settle_balance: 0,
})

const ensureFormState = (item: TokoWorkspaceItem) => {
  if (!formByToko[item.id]) {
    formByToko[item.id] = {
      settlementBalance: '',
      loading: false,
    }
  }
}

const syncForms = () => {
  for (const item of items.value) {
    ensureFormState(item)
  }
}

const workspaceQuery = () => {
  const query: { q?: string; limit: number; offset: number } = {
    limit: pagination.limit,
    offset: pagination.offset,
  }
  if (filters.q.trim() !== '') {
    query.q = filters.q.trim()
  }
  return query
}

const loadWorkspace = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await tokoApi.fetchWorkspace(workspaceQuery())
    workspace.value = page
    pagination.total = page.total
    pagination.limit = page.limit
    pagination.offset = page.offset
    pagination.hasMore = page.has_more
    syncForms()
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const { runNow } = usePolling(loadWorkspace, 12000)

const applyFilters = async () => {
  pagination.offset = 0
  await loadWorkspace()
}

const resetFilters = async () => {
  filters.q = ''
  pagination.offset = 0
  await loadWorkspace()
}

const nextPage = async () => {
  if (!pagination.hasMore) {
    return
  }
  pagination.offset += pagination.limit
  await loadWorkspace()
}

const prevPage = async () => {
  if (pagination.offset <= 0) {
    return
  }
  pagination.offset = Math.max(pagination.offset - pagination.limit, 0)
  await loadWorkspace()
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
    await tokoApi.applySettlement(tokoID, {
      settlement_balance: settlementBalance,
    })
    form.settlementBalance = ''
    toast.success('Settlement berhasil diperbarui')
    await runNow()
  } catch (error) {
    const message = getApiErrorMessage(error)
    errorMessage.value = message
    toast.error(message)
  } finally {
    form.loading = false
  }
}

const resetCreateForm = () => {
  createForm.name = ''
  createForm.callbackURL = ''
}

const resetManageForm = () => {
  manageForm.name = ''
  manageForm.callbackURL = ''
  manageErrorMessage.value = ''
  managedToken.value = ''
}

const openCreateTokoModal = () => {
  if (!canCreateTokoRole.value) {
    createErrorMessage.value = 'Role user tidak memiliki izin membuat toko.'
    return
  }
  createdToko.value = null
  createErrorMessage.value = ''
  resetCreateForm()
  createDialogOpen.value = true
}

const openManageTokoModal = (item: TokoWorkspaceItem) => {
  if (!canCreateTokoRole.value) {
    toast.error('Role user tidak memiliki izin mengelola toko.')
    return
  }

  managedToko.value = item
  manageForm.name = item.name
  manageForm.callbackURL = item.callback_url ?? ''
  managedToken.value = item.token
  manageErrorMessage.value = ''
  manageDialogOpen.value = true
}

const submitCreateToko = async () => {
  if (!canCreateTokoRole.value) {
    createErrorMessage.value = 'Role user tidak memiliki izin membuat toko.'
    return
  }

  createErrorMessage.value = ''
  createLoading.value = true
  try {
    const created = await tokoApi.createToko({
      name: createForm.name.trim(),
      callback_url: createForm.callbackURL.trim() || undefined,
    })
    createdToko.value = created
    toast.success('Toko berhasil dibuat')
    resetCreateForm()
    await runNow()
  } catch (error) {
    const message = getApiErrorMessage(error)
    createErrorMessage.value = message
    toast.error(message)
  } finally {
    createLoading.value = false
  }
}

const submitManageToko = async () => {
  if (!managedToko.value || !canCreateTokoRole.value) {
    return
  }

  manageLoading.value = true
  manageErrorMessage.value = ''
  try {
    const updated = await tokoApi.updateToko(managedToko.value.id, {
      name: manageForm.name.trim(),
      callback_url: manageForm.callbackURL.trim() || undefined,
    })
    managedToken.value = updated.token
    toast.success('Toko berhasil diperbarui')
    await runNow()
  } catch (error) {
    const message = getApiErrorMessage(error)
    manageErrorMessage.value = message
    toast.error(message)
  } finally {
    manageLoading.value = false
  }
}

const submitRegenerateToken = async () => {
  if (!managedToko.value || !canCreateTokoRole.value) {
    return
  }

  regenerateTokenLoading.value = true
  manageErrorMessage.value = ''
  try {
    const updated = await tokoApi.regenerateTokoToken(managedToko.value.id)
    managedToken.value = updated.token
    toast.success('Token toko berhasil diganti')
    await runNow()
  } catch (error) {
    const message = getApiErrorMessage(error)
    manageErrorMessage.value = message
    toast.error(message)
  } finally {
    regenerateTokenLoading.value = false
  }
}

const copyToClipboard = async (value: string, label = 'Token toko') => {
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

const rangeLabel = computed(() => {
  if (pagination.total === 0 || items.value.length === 0) {
    return 'No data'
  }
  const start = pagination.offset + 1
  const end = Math.min(pagination.offset + items.value.length, pagination.total)
  return `Showing ${start}-${end} of ${pagination.total}`
})

const currentPage = computed(() => Math.floor(pagination.offset / pagination.limit) + 1)
const formatDate = (value: string) => formatDateMedium(value)
const formatCurrencyWithDecimals = (value: number) => formatCurrency(value, 2)

void loadWorkspace()
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Toko Workspace"
      title="Manage Toko & Settlement"
      description="Searchable, paginated toko workspace untuk token management dan settlement internal."
      :updated-at="lastUpdated"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="loading" @click="loadWorkspace">
          <Spinner v-if="loading" class="mr-2" />
          Refresh
        </Button>

        <Dialog v-if="canCreateTokoRole" v-model:open="createDialogOpen">
          <DialogTrigger as-child>
            <Button size="sm" @click="openCreateTokoModal">
              <Plus class="mr-2 h-4 w-4" />
              Create Toko
            </Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Create Toko</DialogTitle>
              <DialogDescription>Maksimal 3 toko per creator divalidasi backend. Token dibuat otomatis oleh sistem.</DialogDescription>
            </DialogHeader>

            <div class="grid gap-4 py-2">
              <div class="space-y-2">
                <Label for="toko-name">Nama Toko</Label>
                <Input id="toko-name" v-model="createForm.name" placeholder="Contoh: Toko Alfa" />
              </div>
              <div class="space-y-2">
                <Label for="callback-url">Callback URL (opsional)</Label>
                <Input id="callback-url" v-model="createForm.callbackURL" placeholder="https://domain/callback" />
              </div>
            </div>

            <Alert v-if="createErrorMessage" variant="destructive">
              <TriangleAlert class="h-4 w-4" />
              <AlertTitle>Failed to Create Toko</AlertTitle>
              <AlertDescription>{{ createErrorMessage }}</AlertDescription>
            </Alert>

            <div v-if="createdToko" class="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4">
              <p class="text-sm font-semibold text-emerald-300">Toko berhasil dibuat: {{ createdToko.name }}</p>
              <p class="mt-1 text-xs text-muted-foreground">Gunakan token ini sebagai Bearer token untuk internal payment endpoint.</p>
              <div class="mt-3 flex items-center gap-2 rounded-lg border bg-(--background-muted) px-3 py-2">
                <code class="min-w-0 flex-1 truncate text-xs">{{ createdToko.token }}</code>
                <Button size="sm" variant="outline" type="button" @click="copyToClipboard(createdToko.token, 'Token toko baru')">
                  <Copy class="mr-2 h-4 w-4" />
                  Copy
                </Button>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" :disabled="createLoading" @click="createDialogOpen = false">Close</Button>
              <Button :disabled="createLoading || !canCreateTokoRole" @click="submitCreateToko">
                <Spinner v-if="createLoading" class="mr-2" />
                {{ createLoading ? 'Creating...' : 'Create Toko' }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog v-if="canCreateTokoRole" v-model:open="manageDialogOpen">
          <DialogContent class="sm:max-w-xl">
            <DialogHeader>
              <DialogTitle>Manage Toko</DialogTitle>
              <DialogDescription>Perbarui nama toko, callback URL, dan generate token Bearer baru untuk API project.</DialogDescription>
            </DialogHeader>

            <div class="grid gap-4 py-2">
              <div class="space-y-2">
                <Label for="manage-toko-name">Nama Toko</Label>
                <Input id="manage-toko-name" v-model="manageForm.name" placeholder="Contoh: Toko Alfa" />
              </div>
              <div class="space-y-2">
                <Label for="manage-callback-url">Callback URL</Label>
                <Input id="manage-callback-url" v-model="manageForm.callbackURL" placeholder="https://domain/callback" />
              </div>

              <div class="rounded-xl border bg-(--background-muted) p-4">
                <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div class="space-y-1">
                    <p class="text-sm font-medium text-foreground">API Token</p>
                    <p class="text-xs text-muted-foreground">Gunakan token ini sebagai Bearer token untuk internal payment endpoint.</p>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    :disabled="regenerateTokenLoading"
                    @click="submitRegenerateToken"
                  >
                    <Spinner v-if="regenerateTokenLoading" class="mr-2" />
                    <KeyRound v-else class="mr-2 h-4 w-4" />
                    {{ regenerateTokenLoading ? 'Generating...' : 'Generate New Token' }}
                  </Button>
                </div>

                <div class="mt-3 flex items-center gap-2 rounded-lg border bg-background px-3 py-2">
                  <code class="min-w-0 flex-1 truncate text-xs">{{ managedToken || managedToko?.token || '-' }}</code>
                  <Button
                    size="sm"
                    variant="outline"
                    type="button"
                    :disabled="!(managedToken || managedToko?.token)"
                    @click="copyToClipboard(managedToken || managedToko?.token || '', 'Token toko')"
                  >
                    <Copy class="mr-2 h-4 w-4" />
                    Copy
                  </Button>
                </div>
              </div>
            </div>

            <Alert v-if="manageErrorMessage" variant="destructive">
              <TriangleAlert class="h-4 w-4" />
              <AlertTitle>Failed to Manage Toko</AlertTitle>
              <AlertDescription>{{ manageErrorMessage }}</AlertDescription>
            </Alert>

            <DialogFooter>
              <Button
                variant="outline"
                :disabled="manageLoading || regenerateTokenLoading"
                @click="manageDialogOpen = false; managedToko = null; resetManageForm()"
              >
                Close
              </Button>
              <Button :disabled="manageLoading || regenerateTokenLoading" @click="submitManageToko">
                <Spinner v-if="manageLoading" class="mr-2" />
                {{ manageLoading ? 'Saving...' : 'Save Changes' }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </template>
    </PageHeader>

    <Alert v-if="errorMessage" variant="destructive">
      <TriangleAlert class="h-4 w-4" />
      <AlertTitle>Failed to Load Toko Workspace</AlertTitle>
      <AlertDescription>{{ errorMessage }}</AlertDescription>
    </Alert>

    <div class="grid gap-4 md:grid-cols-3">
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Total Pending Balance</CardDescription>
          <CardTitle class="text-2xl">{{ formatCurrencyWithDecimals(summary.total_pending_balance) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Total Settle Balance</CardDescription>
          <CardTitle class="text-2xl">{{ formatCurrencyWithDecimals(summary.total_settle_balance) }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Visible Toko</CardDescription>
          <CardTitle class="text-2xl">{{ summary.total_tokos }}</CardTitle>
        </CardHeader>
      </Card>
    </div>

    <Card class="app-panel app-filter-card">
      <CardHeader>
        <CardTitle>Filters</CardTitle>
        <CardDescription>Search berdasarkan nama toko, token, atau callback URL. Pagination diproses server-side.</CardDescription>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto]">
          <div class="relative">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              v-model="filters.q"
              class="pl-9"
              placeholder="Search toko name, token, atau callback URL"
              @keydown.enter.prevent="applyFilters"
            />
          </div>
          <Button variant="outline" :disabled="loading" @click="resetFilters">Reset</Button>
          <Button :disabled="loading" @click="applyFilters">Apply Filters</Button>
        </div>
      </CardContent>
    </Card>

    <div v-if="loading && !workspace" class="space-y-4">
      <Card class="app-panel">
        <CardContent class="space-y-3 p-6">
          <Skeleton class="h-8 w-48" />
          <Skeleton class="h-12 w-full" />
          <Skeleton class="h-12 w-full" />
          <Skeleton class="h-12 w-full" />
        </CardContent>
      </Card>
      <Card class="app-panel">
        <CardContent class="space-y-3 p-6">
          <Skeleton class="h-8 w-56" />
          <Skeleton class="h-12 w-full" />
          <Skeleton class="h-12 w-full" />
          <Skeleton class="h-12 w-full" />
        </CardContent>
      </Card>
    </div>

    <template v-else>
      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Toko Management</CardTitle>
          <CardDescription>Token digunakan sebagai Bearer pada internal payment endpoint.</CardDescription>
        </CardHeader>
        <CardContent class="app-table-shell">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Toko</TableHead>
                <TableHead>Charge</TableHead>
                <TableHead>Callback URL</TableHead>
                <TableHead>Token</TableHead>
                <TableHead class="text-right">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <template v-if="items.length > 0">
                <TableRow v-for="item in items" :key="item.id">
                  <TableCell class="font-medium">{{ item.name }}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{{ item.charge }}%</Badge>
                  </TableCell>
                  <TableCell class="max-w-70 truncate text-muted-foreground">
                    {{ item.callback_url || '-' }}
                  </TableCell>
                  <TableCell class="max-w-60">
                    <code class="block truncate rounded-md bg-(--background-muted) px-2 py-1 text-xs">{{ item.token }}</code>
                  </TableCell>
                  <TableCell class="text-right">
                    <div class="flex justify-end gap-2">
                      <Button size="sm" variant="outline" type="button" @click="copyToClipboard(item.token)">
                        <Copy class="mr-2 h-4 w-4" />
                        Copy Token
                      </Button>
                      <Button
                        v-if="canCreateTokoRole"
                        size="sm"
                        variant="outline"
                        type="button"
                        @click="openManageTokoModal(item)"
                      >
                        <PencilLine class="mr-2 h-4 w-4" />
                        Manage
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              </template>
              <TableEmpty v-else :colspan="5">
                <EmptyState
                  title="Belum Ada Toko"
                  description="Tambahkan toko baru atau ubah filter untuk melihat data."
                  :action-label="canCreateTokoRole ? 'Create Toko' : undefined"
                  @action="openCreateTokoModal"
                />
              </TableEmpty>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card class="app-panel">
        <CardHeader>
          <CardTitle>Pending & Settle Balances</CardTitle>
          <CardDescription>Pending bertambah dari transaksi sukses. Settlement manual developer memindahkan pending ke settle balance.</CardDescription>
        </CardHeader>
        <CardContent class="app-table-shell">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Toko</TableHead>
                <TableHead class="text-right">Pending Balance</TableHead>
                <TableHead class="text-right">Settle Balance</TableHead>
                <TableHead>Updated</TableHead>
                <TableHead class="text-right">Settlement Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <template v-if="items.length > 0">
                <TableRow v-for="item in items" :key="`balance-${item.id}`">
                  <TableCell class="font-medium">{{ item.name }}</TableCell>
                  <TableCell class="text-right">{{ formatCurrencyWithDecimals(item.pending_balance) }}</TableCell>
                  <TableCell class="text-right">{{ formatCurrencyWithDecimals(item.settle_balance) }}</TableCell>
                  <TableCell>{{ formatDate(item.updated_at) }}</TableCell>
                  <TableCell class="text-right">
                    <form
                      v-if="canManualSettlement"
                      class="flex flex-col items-end gap-2 md:flex-row md:justify-end"
                      @submit.prevent="submitSettlement(item.id)"
                    >
                      <Input
                        v-model="formByToko[item.id].settlementBalance"
                        class="w-full md:w-47.5"
                        type="number"
                        step="0.01"
                        min="0"
                        placeholder="Settlement amount"
                      />
                      <Button
                        type="submit"
                        size="sm"
                        :disabled="formByToko[item.id].loading"
                      >
                        <Spinner v-if="formByToko[item.id].loading" class="mr-2" />
                        {{ formByToko[item.id].loading ? 'Saving...' : 'Apply' }}
                      </Button>
                    </form>
                    <span v-else class="text-sm text-muted-foreground">Developer only</span>
                  </TableCell>
                </TableRow>
              </template>
              <TableEmpty v-else :colspan="5">
                <EmptyState
                  title="Belum Ada Balance Toko"
                  description="Balance settlement akan muncul otomatis setelah toko tersedia."
                />
              </TableEmpty>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <div class="app-pagination-bar">
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ rangeLabel }}</p>
          <p class="text-xs text-muted-foreground">Page {{ currentPage }} • Limit {{ pagination.limit }}</p>
        </div>
        <div class="flex items-center gap-2">
          <Button size="sm" variant="outline" :disabled="loading || pagination.offset <= 0" @click="prevPage">
            Prev
          </Button>
          <Button size="sm" variant="outline" :disabled="loading || !pagination.hasMore" @click="nextPage">
            Next
          </Button>
        </div>
      </div>
    </template>
  </section>
</template>
