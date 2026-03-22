<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Plus, TriangleAlert } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getApiErrorMessage } from '@/services/http'
import type { User, UserRole } from '@/services/types'
import * as userApi from '@/services/user'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const route = useRoute()
const router = useRouter()
const users = ref<User[]>([])
const loading = ref(false)
const errorMessage = ref('')
const createDialogOpen = ref(false)
const createLoading = ref(false)
const roleSaveLoading = ref<Record<number, boolean>>({})

const createForm = reactive({
  name: '',
  email: '',
  password: '',
  role: 'user' as UserRole,
  is_active: true,
})

const roleDraftByUser = reactive<Record<number, UserRole>>({})

const actorRole = computed<UserRole>(() => userStore.profile?.role ?? 'user')
const canEditRole = computed(() => actorRole.value === 'dev' || actorRole.value === 'superadmin')

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

const loadUsers = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await userApi.list(200)
    users.value = result
    for (const user of result) {
      roleDraftByUser[user.id] = user.role
    }
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const resetCreateForm = () => {
  createForm.name = ''
  createForm.email = ''
  createForm.password = ''
  createForm.role = assignableRoles.value[0] ?? 'user'
  createForm.is_active = true
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

const statusBadgeVariant = (isActive: boolean) => (isActive ? 'default' : 'secondary')

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

void loadUsers()
</script>

<template>
  <section class="page-shell">
    <PageHeader
      eyebrow="Administration"
      title="User Management"
      description="Kelola user, role, dan status aktif dengan kontrol RBAC."
    >
      <template #actions>
        <Button variant="outline" :disabled="loading" @click="loadUsers">
          <Spinner v-if="loading" class="mr-2" />
          Refresh
        </Button>
        <Dialog v-model:open="createDialogOpen">
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

    <div v-if="loading" class="space-y-3 rounded-xl border bg-[var(--background-elevated)] p-4">
      <Skeleton class="h-8 w-64" />
      <Skeleton class="h-12 w-full" />
      <Skeleton class="h-12 w-full" />
      <Skeleton class="h-12 w-full" />
    </div>

    <div v-else-if="users.length === 0">
      <EmptyState
        title="Belum Ada User"
        description="Tambahkan user pertama untuk mulai mendelegasikan akses."
        action-label="Add User"
        @action="createDialogOpen = true"
      />
    </div>

    <div v-else class="app-panel app-table-shell p-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Status</TableHead>
            <TableHead v-if="canEditRole" class="text-right">Action</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="user in users" :key="user.id">
            <TableCell class="font-medium">{{ user.name }}</TableCell>
            <TableCell>{{ user.email }}</TableCell>
            <TableCell>
              <div class="flex items-center gap-2">
                <Badge variant="outline" class="capitalize">{{ user.role }}</Badge>
                <Select
                  v-if="canEditRole && user.role !== 'dev'"
                  v-model="roleDraftByUser[user.id]"
                >
                  <SelectTrigger class="w-[150px]">
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
              <Badge :variant="statusBadgeVariant(user.is_active)">{{ user.is_active ? 'Active' : 'Inactive' }}</Badge>
            </TableCell>
            <TableCell v-if="canEditRole" class="text-right">
              <Button
                v-if="user.role !== 'dev'"
                size="sm"
                variant="outline"
                :disabled="roleSaveLoading[user.id] || roleDraftByUser[user.id] === user.role"
                @click="saveRole(user.id)"
              >
                <Spinner v-if="roleSaveLoading[user.id]" class="mr-2" />
                Save Role
              </Button>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  </section>
</template>
