<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Copy, Plus, Search, Trash2, TriangleAlert } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import { Switch } from '@/components/ui/switch'
import { getApiErrorMessage } from '@/services/http'
import type { User, UserRole } from '@/services/types'
import * as userApi from '@/services/user'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const page = ref<{ items: User[]; total: number; limit: number; offset: number; has_more: boolean } | null>(null)
const loading = ref(false)
const errorMessage = ref('')
const createDialogOpen = ref(false)
const createLoading = ref(false)
const roleSaveLoading = ref<Record<number, boolean>>({})
const statusSaveLoading = ref<Record<number, boolean>>({})
const deleteDialogOpen = ref(false)
const deleteLoading = ref(false)
const pendingDeleteUser = ref<User | null>(null)

const createForm = reactive({
  name: '',
  email: '',
  password: '',
  role: 'user' as UserRole,
  is_active: true,
})

const filters = reactive({
  q: '',
  role: 'all' as 'all' | UserRole,
})

const pagination = reactive({
  limit: 10,
  offset: 0,
  total: 0,
  hasMore: false,
})

const roleDraftByUser = reactive<Record<number, UserRole>>({})
const actorUserID = computed(() => userStore.profile?.id ?? 0)
const actorRole = computed<UserRole>(() => userStore.profile?.role ?? 'user')
const canEditRole = computed(() => actorRole.value === 'dev' || actorRole.value === 'superadmin')
const users = computed(() => page.value?.items ?? [])

const assignableRoles = computed<UserRole[]>(() => {
  if (actorRole.value === 'dev') {
    return ['superadmin', 'admin', 'user']
  }
  if (actorRole.value === 'superadmin') {
    return ['admin', 'user']
  }
  if (actorRole.value === 'admin') {
    return ['user']
  }
  return ['user']
})

const filterRoleOptions = computed<Array<'all' | UserRole>>(() => ['all', 'dev', 'superadmin', 'admin', 'user'])

const canManageTargetUser = (target: User) => {
  if (actorRole.value === 'user' || target.id === actorUserID.value) {
    return false
  }

  switch (actorRole.value) {
    case 'dev':
      return target.role !== 'dev'
    case 'superadmin':
      return target.role === 'admin' || target.role === 'user'
    case 'admin':
      return target.role === 'user'
    default:
      return false
  }
}

const listQuery = () => ({
  limit: pagination.limit,
  offset: pagination.offset,
  q: filters.q.trim() || undefined,
  role: filters.role === 'all' ? undefined : filters.role,
})

const loadUsers = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await userApi.list(listQuery())
    page.value = result
    pagination.total = result.total
    pagination.limit = result.limit
    pagination.offset = result.offset
    pagination.hasMore = result.has_more
    for (const user of result.items) {
      roleDraftByUser[user.id] = user.role
    }
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const replaceUserInPage = (updatedUser: User) => {
  if (!page.value) {
    return
  }

  page.value = {
    ...page.value,
    items: page.value.items.map((user) => (user.id === updatedUser.id ? updatedUser : user)),
  }
  roleDraftByUser[updatedUser.id] = updatedUser.role
}

const resetCreateForm = () => {
  createForm.name = ''
  createForm.email = ''
  createForm.password = ''
  createForm.role = assignableRoles.value[0] ?? 'user'
  createForm.is_active = true
}

const applyFilters = async () => {
  pagination.offset = 0
  await loadUsers()
}

const resetFilters = async () => {
  filters.q = ''
  filters.role = 'all'
  pagination.offset = 0
  await loadUsers()
}

const nextPage = async () => {
  if (!pagination.hasMore) {
    return
  }
  pagination.offset += pagination.limit
  await loadUsers()
}

const prevPage = async () => {
  if (pagination.offset <= 0) {
    return
  }
  pagination.offset = Math.max(pagination.offset - pagination.limit, 0)
  await loadUsers()
}

const createUser = async () => {
  createLoading.value = true
  try {
    await userApi.create({
      name: createForm.name.trim(),
      email: createForm.email.trim(),
      password: createForm.password,
      role: createForm.role,
      is_active: createForm.is_active,
    })
    toast.success('User berhasil dibuat')
    createDialogOpen.value = false
    resetCreateForm()
    pagination.offset = 0
    await loadUsers()
  } catch (error) {
    toast.error(getApiErrorMessage(error))
  } finally {
    createLoading.value = false
  }
}

const saveRole = async (userID: number) => {
  roleSaveLoading.value[userID] = true
  try {
    await userApi.updateRole(userID, { role: roleDraftByUser[userID] })
    toast.success('Role user berhasil diperbarui')
    await loadUsers()
  } catch (error) {
    toast.error(getApiErrorMessage(error))
  } finally {
    roleSaveLoading.value[userID] = false
  }
}

const toggleUserActive = async (user: User, isActive: boolean) => {
  statusSaveLoading.value[user.id] = true
  try {
    const updated = await userApi.updateActive(user.id, { is_active: isActive })
    replaceUserInPage(updated)
    toast.success(`Status ${updated.name} berhasil diperbarui`)
  } catch (error) {
    toast.error(getApiErrorMessage(error))
  } finally {
    statusSaveLoading.value[user.id] = false
  }
}

const promptDeleteUser = (user: User) => {
  pendingDeleteUser.value = user
  deleteDialogOpen.value = true
}

const confirmDeleteUser = async () => {
  if (!pendingDeleteUser.value) {
    return
  }

  deleteLoading.value = true
  try {
    const target = pendingDeleteUser.value
    await userApi.remove(target.id)
    toast.success(`User ${target.name} berhasil dihapus`)
    deleteDialogOpen.value = false
    pendingDeleteUser.value = null

    if (users.value.length === 1 && pagination.offset > 0) {
      pagination.offset = Math.max(pagination.offset - pagination.limit, 0)
    }

    await loadUsers()
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

const statusBadgeVariant = (isActive: boolean) => (isActive ? 'default' : 'secondary')

const rangeLabel = computed(() => {
  if (pagination.total === 0 || users.value.length === 0) {
    return 'No data'
  }
  const start = pagination.offset + 1
  const end = Math.min(pagination.offset + users.value.length, pagination.total)
  return `Showing ${start}-${end} of ${pagination.total}`
})

const currentPage = computed(() => Math.floor(pagination.offset / pagination.limit) + 1)

watch(
  () => route.query.create,
  (value) => {
    if (value === '1') {
      createDialogOpen.value = true
      void router.replace({ query: { ...route.query, create: undefined } })
    }
  },
  { immediate: true },
)

watch(deleteDialogOpen, (open) => {
  if (!open && !deleteLoading.value) {
    pendingDeleteUser.value = null
  }
})

void loadUsers()
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Administration"
      title="User Management"
      description="Kelola user, role, dan status aktif dengan search, role filter, dan pagination server-side."
    >
      <template #actions>
        <Dialog v-if="actorRole !== 'user'" v-model:open="createDialogOpen">
          <DialogTrigger as-child>
            <Button>
              <Plus class="mr-2 h-4 w-4" />
              Add User
            </Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>Create User</DialogTitle>
              <DialogDescription>Role yang tersedia mengikuti hak akses akun saat ini.</DialogDescription>
            </DialogHeader>

            <div class="grid gap-4 py-2">
              <div class="space-y-2">
                <Label for="create-name">Name</Label>
                <Input id="create-name" v-model="createForm.name" placeholder="Nama lengkap" />
              </div>
              <div class="space-y-2">
                <Label for="create-email">Email</Label>
                <Input id="create-email" v-model="createForm.email" type="email" placeholder="name@company.com" />
              </div>
              <div class="space-y-2">
                <Label for="create-password">Password</Label>
                <Input id="create-password" v-model="createForm.password" type="password" placeholder="Minimal 8 karakter" />
              </div>
              <div class="space-y-2">
                <Label>Role</Label>
                <Select v-model="createForm.role">
                  <SelectTrigger>
                    <SelectValue placeholder="Pilih role" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="role in assignableRoles" :key="role" :value="role">
                      <span class="capitalize">{{ role }}</span>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="flex items-center gap-2">
                <Checkbox id="create-user-active" v-model:checked="createForm.is_active" />
                <Label for="create-user-active" class="text-sm font-normal">
                  User langsung aktif setelah dibuat
                </Label>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" :disabled="createLoading" @click="createDialogOpen = false">Cancel</Button>
              <Button :disabled="createLoading" @click="createUser">
                <Spinner v-if="createLoading" class="mr-2" />
                {{ createLoading ? 'Creating...' : 'Create User' }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </template>
    </PageHeader>

    <Alert v-if="errorMessage" variant="destructive">
      <TriangleAlert class="h-4 w-4" />
      <AlertTitle>Failed to Load Users</AlertTitle>
      <AlertDescription>{{ errorMessage }}</AlertDescription>
    </Alert>

    <Card class="app-panel app-filter-card">
      <CardHeader>
        <CardTitle>Filters</CardTitle>
        <CardDescription>Search berdasarkan nama atau email. Filter role memakai komponen select shadcn-vue.</CardDescription>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px_auto_auto]">
          <div class="relative">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              v-model="filters.q"
              class="pl-9"
              placeholder="Search name atau email"
              @keydown.enter.prevent="applyFilters"
            />
          </div>
          <Select v-model="filters.role">
            <SelectTrigger>
              <SelectValue placeholder="Filter role" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="role in filterRoleOptions" :key="role" :value="role">
                <span class="capitalize">{{ role === 'all' ? 'All Roles' : role }}</span>
              </SelectItem>
            </SelectContent>
          </Select>
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
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Status</TableHead>
            <TableHead class="text-right">Action</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="users.length > 0">
            <TableRow v-for="user in users" :key="user.id">
              <TableCell class="font-medium">{{ user.name }}</TableCell>
              <TableCell>
                <div class="flex items-center gap-2">
                  <span class="truncate">{{ user.email }}</span>
                  <Button size="icon" variant="ghost" class="h-8 w-8 shrink-0" @click="copyToClipboard(user.email, 'Email user')">
                    <Copy class="h-4 w-4" />
                    <span class="sr-only">Copy email</span>
                  </Button>
                </div>
              </TableCell>
              <TableCell>
                <div class="flex items-center gap-2">
                  <Badge variant="outline" class="capitalize">{{ user.role }}</Badge>
                  <Select
                    v-if="canEditRole && user.role !== 'dev'"
                    v-model="roleDraftByUser[user.id]"
                  >
                    <SelectTrigger class="w-37.5">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="role in assignableRoles" :key="`${user.id}-${role}`" :value="role">
                        <span class="capitalize">{{ role }}</span>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </TableCell>
              <TableCell>
                <div class="flex items-center gap-3">
                  <Switch
                    v-if="canManageTargetUser(user)"
                    :model-value="user.is_active"
                    :disabled="statusSaveLoading[user.id]"
                    :aria-label="`Toggle active status for ${user.name}`"
                    @update:model-value="(value) => toggleUserActive(user, Boolean(value))"
                  />
                  <Spinner v-if="statusSaveLoading[user.id]" class="h-4 w-4" />
                  <Badge :variant="statusBadgeVariant(user.is_active)">
                    {{ user.is_active ? 'Active' : 'Inactive' }}
                  </Badge>
                </div>
              </TableCell>
              <TableCell class="text-right">
                <div class="flex justify-end gap-2">
                  <Button size="sm" variant="outline" @click="copyToClipboard(user.email, 'Email user')">
                    Copy Email
                  </Button>
                  <Button
                    v-if="canEditRole && user.role !== 'dev'"
                    size="sm"
                    variant="outline"
                    :disabled="roleSaveLoading[user.id] || roleDraftByUser[user.id] === user.role"
                    @click="saveRole(user.id)"
                  >
                    <Spinner v-if="roleSaveLoading[user.id]" class="mr-2" />
                    Save Role
                  </Button>
                  <Button
                    v-if="canManageTargetUser(user)"
                    size="sm"
                    variant="outline"
                    class="text-destructive hover:text-destructive"
                    @click="promptDeleteUser(user)"
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
              title="Belum Ada User"
              description="Tambahkan user pertama atau ubah filter untuk melihat hasil."
              action-label="Add User"
              @action="createDialogOpen = true"
            />
          </TableEmpty>
        </TableBody>
      </Table>
    </div>

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

    <Dialog v-model:open="deleteDialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete User</DialogTitle>
          <DialogDescription>
            User
            <span class="font-medium text-foreground">{{ pendingDeleteUser?.name }}</span>
            akan dihapus dari sistem ini. Tindakan ini tidak dapat dibatalkan.
          </DialogDescription>
        </DialogHeader>

        <div class="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-muted-foreground">
          Pastikan user ini memang tidak lagi membutuhkan akses ke dashboard dan API project.
        </div>

        <DialogFooter>
          <Button variant="outline" :disabled="deleteLoading" @click="deleteDialogOpen = false">Cancel</Button>
          <Button :disabled="deleteLoading" class="bg-destructive text-destructive-foreground hover:bg-destructive/90" @click="confirmDeleteUser">
            <Spinner v-if="deleteLoading" class="mr-2" />
            {{ deleteLoading ? 'Deleting...' : 'Delete User' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </section>
</template>
