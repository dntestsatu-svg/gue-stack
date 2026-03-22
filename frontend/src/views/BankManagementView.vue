<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Copy, Landmark, Plus, Search, Trash2, TriangleAlert } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import BankCatalogSelect from '@/components/BankCatalogSelect.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
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
import { getApiErrorMessage } from '@/services/http'
import * as bankApi from '@/services/bank'
import type { BankItem, BankListPage, BankPaymentOption } from '@/services/types'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { formatDateMedium } = useFormatters()

const page = ref<BankListPage | null>(null)
const loading = ref(false)
const errorMessage = ref('')
const lastUpdated = ref('')

const createDialogOpen = ref(false)
const createLoading = ref(false)
const createErrorMessage = ref('')

const deleteDialogOpen = ref(false)
const deleteLoading = ref(false)
const pendingDeleteBank = ref<BankItem | null>(null)

const filters = reactive({
  q: '',
})

const pagination = reactive({
  limit: 10,
  offset: 0,
  total: 0,
  hasMore: false,
})

const createForm = reactive({
  paymentID: null as number | null,
  paymentName: '',
  accountName: '',
  accountNumber: '',
})

const canManageBanks = computed(() => {
  const role = userStore.profile?.role
  return role === 'dev' || role === 'superadmin' || role === 'admin'
})

const items = computed(() => page.value?.items ?? [])
const totalBanks = computed(() => page.value?.total ?? 0)
const rangeLabel = computed(() => {
  if (pagination.total === 0 || items.value.length === 0) {
    return 'No data'
  }
  const start = pagination.offset + 1
  const end = Math.min(pagination.offset + items.value.length, pagination.total)
  return `Showing ${start}-${end} of ${pagination.total}`
})
const currentPage = computed(() => Math.floor(pagination.offset / pagination.limit) + 1)

const listQuery = () => ({
  limit: pagination.limit,
  offset: pagination.offset,
  q: filters.q.trim() || undefined,
})

const loadBanks = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await bankApi.list(listQuery())
    page.value = result
    pagination.total = result.total
    pagination.limit = result.limit
    pagination.offset = result.offset
    pagination.hasMore = result.has_more
    lastUpdated.value = new Date().toISOString()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const applyFilters = async () => {
  pagination.offset = 0
  await loadBanks()
}

const resetFilters = async () => {
  filters.q = ''
  pagination.offset = 0
  await loadBanks()
}

const nextPage = async () => {
  if (!pagination.hasMore) {
    return
  }
  pagination.offset += pagination.limit
  await loadBanks()
}

const prevPage = async () => {
  if (pagination.offset <= 0) {
    return
  }
  pagination.offset = Math.max(pagination.offset - pagination.limit, 0)
  await loadBanks()
}

const resetCreateForm = () => {
  createForm.paymentID = null
  createForm.paymentName = ''
  createForm.accountName = ''
  createForm.accountNumber = ''
  createErrorMessage.value = ''
}

const handlePaymentSelect = (option: BankPaymentOption) => {
  createForm.paymentID = option.id
  createForm.paymentName = option.bank_name
}

const createBank = async () => {
  if (!canManageBanks.value) {
    createErrorMessage.value = 'Role user tidak memiliki izin mengelola bank.'
    return
  }
  if (!createForm.paymentID) {
    createErrorMessage.value = 'Pilih bank terlebih dahulu dari payment catalog.'
    return
  }

  createLoading.value = true
  createErrorMessage.value = ''
  try {
    await bankApi.create({
      payment_id: createForm.paymentID,
      account_name: createForm.accountName.trim(),
      account_number: createForm.accountNumber.trim(),
    })
    toast.success('Bank berhasil disimpan')
    createDialogOpen.value = false
    resetCreateForm()
    pagination.offset = 0
    await loadBanks()
  } catch (error) {
    const message = getApiErrorMessage(error)
    createErrorMessage.value = message
    toast.error(message)
  } finally {
    createLoading.value = false
  }
}

const promptDeleteBank = (bank: BankItem) => {
  pendingDeleteBank.value = bank
  deleteDialogOpen.value = true
}

const confirmDeleteBank = async () => {
  if (!pendingDeleteBank.value) {
    return
  }

  deleteLoading.value = true
  try {
    const target = pendingDeleteBank.value
    await bankApi.remove(target.id)
    toast.success(`Bank ${target.bank_name} berhasil dihapus`)
    deleteDialogOpen.value = false
    pendingDeleteBank.value = null

    if (items.value.length === 1 && pagination.offset > 0) {
      pagination.offset = Math.max(pagination.offset - pagination.limit, 0)
    }

    await loadBanks()
  } catch (error) {
    toast.error(getApiErrorMessage(error))
  } finally {
    deleteLoading.value = false
  }
}

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

void loadBanks()
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Bank Workspace"
      title="Bank Management"
      description="Kelola rekening payout user login dengan validasi bank terhadap payment catalog dan pencarian server-side."
      :updated-at="lastUpdated"
    >
      <template #actions>
        <Dialog v-if="canManageBanks" v-model:open="createDialogOpen">
          <DialogTrigger as-child>
            <Button @click="resetCreateForm">
              <Plus class="mr-2 h-4 w-4" />
              Add Bank
            </Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-xl">
            <DialogHeader>
              <DialogTitle>Add Bank</DialogTitle>
              <DialogDescription>
                Bank name dipilih dari payment catalog. Backend akan memvalidasi bank_code dan bank_name berdasarkan data payments.
              </DialogDescription>
            </DialogHeader>

            <div class="grid gap-4 py-2">
              <div class="space-y-2">
                <Label>Bank Name</Label>
                <BankCatalogSelect
                  v-model="createForm.paymentID"
                  :selected-label="createForm.paymentName"
                  @select="handlePaymentSelect"
                />
              </div>

              <div class="space-y-2">
                <Label for="bank-account-name">Account Name</Label>
                <Input
                  id="bank-account-name"
                  v-model="createForm.accountName"
                  placeholder="Nama pemilik rekening"
                />
              </div>

              <div class="space-y-2">
                <Label for="bank-account-number">Account Number</Label>
                <Input
                  id="bank-account-number"
                  v-model="createForm.accountNumber"
                  inputmode="numeric"
                  placeholder="Nomor rekening"
                />
              </div>
            </div>

            <Alert v-if="createErrorMessage" variant="destructive">
              <TriangleAlert class="h-4 w-4" />
              <AlertTitle>Failed to Create Bank</AlertTitle>
              <AlertDescription>{{ createErrorMessage }}</AlertDescription>
            </Alert>

            <DialogFooter>
              <Button variant="outline" :disabled="createLoading" @click="createDialogOpen = false">Cancel</Button>
              <Button :disabled="createLoading" @click="createBank">
                <Spinner v-if="createLoading" class="mr-2" />
                {{ createLoading ? 'Saving...' : 'Save Bank' }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </template>
    </PageHeader>

    <Alert v-if="errorMessage" variant="destructive">
      <TriangleAlert class="h-4 w-4" />
      <AlertTitle>Failed to Load Banks</AlertTitle>
      <AlertDescription>{{ errorMessage }}</AlertDescription>
    </Alert>

    <div class="grid gap-4 md:grid-cols-2">
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Saved Bank Accounts</CardDescription>
          <CardTitle class="text-2xl">{{ totalBanks }}</CardTitle>
        </CardHeader>
      </Card>
      <Card class="dashboard-kpi-card">
        <CardHeader class="pb-2">
          <CardDescription>Current Scope</CardDescription>
          <CardTitle class="text-2xl">User Owned</CardTitle>
        </CardHeader>
      </Card>
    </div>

    <Card class="app-panel">
      <CardHeader>
        <CardTitle>Filters</CardTitle>
        <CardDescription>Search berdasarkan bank name, account name, atau account number. Pagination diproses server-side.</CardDescription>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto]">
          <div class="relative">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              v-model="filters.q"
              class="pl-9"
              placeholder="Search bank name, account name, atau number"
              @keydown.enter.prevent="applyFilters"
            />
          </div>
          <Button variant="outline" :disabled="loading" @click="resetFilters">Reset</Button>
          <Button :disabled="loading" @click="applyFilters">Apply Filters</Button>
        </div>
      </CardContent>
    </Card>

    <div v-if="loading && !page" class="space-y-3 rounded-xl border bg-(--background-elevated) p-4">
      <Skeleton class="h-8 w-64" />
      <Skeleton class="h-12 w-full" />
      <Skeleton class="h-12 w-full" />
      <Skeleton class="h-12 w-full" />
    </div>

    <div v-else class="app-panel app-table-shell p-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Bank</TableHead>
            <TableHead>Account Name</TableHead>
            <TableHead>Account Number</TableHead>
            <TableHead>Added</TableHead>
            <TableHead class="text-right">Action</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="items.length > 0">
            <TableRow v-for="bank in items" :key="bank.id">
              <TableCell class="font-medium">
                <div class="flex items-center gap-2">
                  <Landmark class="h-4 w-4 text-primary" />
                  <span>{{ bank.bank_name }}</span>
                </div>
              </TableCell>
              <TableCell>{{ bank.account_name }}</TableCell>
              <TableCell>
                <div class="flex items-center gap-2">
                  <span class="truncate">{{ bank.account_number }}</span>
                  <Button size="icon" variant="ghost" class="h-8 w-8 shrink-0" @click="copyToClipboard(bank.account_number, 'Nomor rekening')">
                    <Copy class="h-4 w-4" />
                    <span class="sr-only">Copy account number</span>
                  </Button>
                </div>
              </TableCell>
              <TableCell>{{ formatDateMedium(bank.created_at) }}</TableCell>
              <TableCell class="text-right">
                <div class="flex justify-end gap-2">
                  <Button size="sm" variant="outline" @click="copyToClipboard(bank.account_number, 'Nomor rekening')">
                    <Copy class="mr-2 h-4 w-4" />
                    Copy
                  </Button>
                  <Button
                    v-if="canManageBanks"
                    size="sm"
                    variant="outline"
                    class="text-destructive hover:text-destructive"
                    @click="promptDeleteBank(bank)"
                  >
                    <Trash2 class="mr-2 h-4 w-4" />
                    Delete
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </template>
          <TableEmpty v-else :colspan="5">
            <EmptyState
              title="Belum Ada Bank"
              description="Tambahkan rekening pertama agar payout dan operasional bank user ini tersimpan rapi."
              :action-label="canManageBanks ? 'Add Bank' : undefined"
              @action="createDialogOpen = true"
            />
          </TableEmpty>
        </TableBody>
      </Table>
    </div>

    <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border bg-(--background-elevated) px-4 py-3">
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

    <Dialog v-model:open="deleteDialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete Bank</DialogTitle>
          <DialogDescription>
            Rekening
            <span class="font-medium text-foreground">{{ pendingDeleteBank?.bank_name }}</span>
            akan dihapus dari akun ini. Tindakan ini tidak dapat dibatalkan.
          </DialogDescription>
        </DialogHeader>

        <div class="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-muted-foreground">
          Pastikan rekening ini memang tidak lagi digunakan untuk payout atau settlement pada workflow user ini.
        </div>

        <DialogFooter>
          <Button variant="outline" :disabled="deleteLoading" @click="deleteDialogOpen = false">Cancel</Button>
          <Button :disabled="deleteLoading" class="bg-destructive text-destructive-foreground hover:bg-destructive/90" @click="confirmDeleteBank">
            <Spinner v-if="deleteLoading" class="mr-2" />
            {{ deleteLoading ? 'Deleting...' : 'Delete Bank' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </section>
</template>
