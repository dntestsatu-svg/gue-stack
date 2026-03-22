<script setup lang="ts">
import { reactive, ref } from 'vue'
import { KeyRound, ShieldAlert } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { initCsrf } from '@/services/auth'
import { getApiErrorMessage } from '@/services/http'
import { changePassword } from '@/services/user'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
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
import { Spinner } from '@/components/ui/spinner'

const open = ref(false)
const submitting = ref(false)
const formError = ref('')
const form = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})

function resetForm() {
  form.currentPassword = ''
  form.newPassword = ''
  form.confirmPassword = ''
  formError.value = ''
}

function validateForm() {
  if (!form.currentPassword || !form.newPassword || !form.confirmPassword) {
    return 'Semua field password wajib diisi.'
  }
  if (form.newPassword.length < 8) {
    return 'Password baru minimal 8 karakter.'
  }
  if (form.newPassword !== form.confirmPassword) {
    return 'Konfirmasi password baru belum sama.'
  }
  if (form.currentPassword === form.newPassword) {
    return 'Password baru harus berbeda dari password saat ini.'
  }
  return ''
}

async function submit() {
  const validationError = validateForm()
  formError.value = validationError
  if (validationError) {
    return
  }

  submitting.value = true
  try {
    await initCsrf()
    const message = await changePassword({
      current_password: form.currentPassword,
      new_password: form.newPassword,
      confirm_password: form.confirmPassword,
    })
    toast.success(message)
    open.value = false
    resetForm()
  } catch (error) {
    formError.value = getApiErrorMessage(error)
  } finally {
    submitting.value = false
  }
}

function handleOpenChange(nextOpen: boolean) {
  open.value = nextOpen
  if (!nextOpen) {
    resetForm()
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="handleOpenChange">
    <DialogTrigger as-child>
      <Button variant="outline" size="icon" class="hidden md:inline-flex">
        <KeyRound class="h-4 w-4" />
        <span class="sr-only">Change password</span>
      </Button>
    </DialogTrigger>

    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Ubah Password</DialogTitle>
        <DialogDescription>
          Perbarui password akun yang sedang login dengan aman. Password baru akan langsung aktif setelah berhasil disimpan.
        </DialogDescription>
      </DialogHeader>

      <form class="space-y-4" @submit.prevent="submit">
        <Alert v-if="formError" variant="destructive">
          <ShieldAlert class="h-4 w-4" />
          <AlertTitle>Perubahan password belum berhasil</AlertTitle>
          <AlertDescription>{{ formError }}</AlertDescription>
        </Alert>

        <div class="space-y-2">
          <Label for="current-password">Password saat ini</Label>
          <Input
            id="current-password"
            v-model="form.currentPassword"
            type="password"
            autocomplete="current-password"
            placeholder="Masukkan password saat ini"
          />
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-2">
            <Label for="new-password">Password baru</Label>
            <Input
              id="new-password"
              v-model="form.newPassword"
              type="password"
              autocomplete="new-password"
              placeholder="Minimal 8 karakter"
            />
          </div>

          <div class="space-y-2">
            <Label for="confirm-password">Konfirmasi password baru</Label>
            <Input
              id="confirm-password"
              v-model="form.confirmPassword"
              type="password"
              autocomplete="new-password"
              placeholder="Ulangi password baru"
            />
          </div>
        </div>

        <DialogFooter class="gap-2 sm:justify-end">
          <Button type="button" variant="outline" :disabled="submitting" @click="handleOpenChange(false)">
            Batal
          </Button>
          <Button type="submit" :disabled="submitting">
            <Spinner v-if="submitting" class="mr-2 h-4 w-4" />
            <span>{{ submitting ? 'Menyimpan...' : 'Simpan Password' }}</span>
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
