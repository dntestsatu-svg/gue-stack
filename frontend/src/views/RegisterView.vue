<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { TriangleAlert } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { ensureGtmLoaded, pushGtmEvent } from '@/lib/gtm'

const auth = useAuthStore()
const router = useRouter()
const errorMessage = ref('')

const form = reactive({
  name: '',
  email: '',
  password: '',
})

const redirectToDashboard = () => {
  window.location.assign('/dashboard')
}

const onSubmit = async () => {
  errorMessage.value = ''
  try {
    await auth.register({
      name: form.name.trim(),
      email: form.email.trim(),
      password: form.password,
    })

    const gtmTracked = pushGtmEvent({
      event: 'sign_up',
      method: 'email',
      account_role: 'admin',
      page_type: 'register',
      eventCallback: redirectToDashboard,
      eventTimeout: 800,
    })

    if (gtmTracked) {
      window.setTimeout(redirectToDashboard, 900)
      return
    }

    await router.push('/dashboard')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Registrasi gagal'
  }
}

onMounted(() => {
  ensureGtmLoaded()
})
</script>

<template>
  <section class="app-auth-card-grid">
    <div class="app-auth-copy">
      <p class="app-auth-kicker">Workspace Enrollment</p>
      <h1 class="app-auth-title">Buat akun baru untuk mulai mengelola toko, transaksi, dan saldo secara terpusat.</h1>
      <p class="app-auth-description">
        Gunakan akun ini untuk masuk ke dashboard operasional, mengelola integrasi merchant, dan memantau aliran
        transaksi secara real-time.
      </p>

      <div class="app-auth-points">
        <div class="app-auth-point">
          <strong>Merchant Ready</strong>
          <span>Dokumentasi API, token toko, callback readiness, dan testing tersedia dalam satu workspace.</span>
        </div>
        <div class="app-auth-point">
          <strong>Finance Aware</strong>
          <span>Pending, settle, withdraw, dan platform fee mengikuti aturan yang konsisten dan terukur.</span>
        </div>
      </div>
    </div>

    <Card class="app-auth-card">
      <CardHeader class="space-y-2">
        <CardTitle class="text-2xl tracking-tight">Create Account</CardTitle>
        <CardDescription>Daftar untuk mulai menggunakan API gateway workspace.</CardDescription>
      </CardHeader>

      <CardContent class="space-y-5">
        <Alert v-if="errorMessage" variant="destructive">
          <TriangleAlert class="h-4 w-4" />
          <AlertTitle>Registration Error</AlertTitle>
          <AlertDescription>{{ errorMessage }}</AlertDescription>
        </Alert>

        <form class="app-auth-form" @submit.prevent="onSubmit">
          <div class="space-y-2">
            <Label for="name">Full Name</Label>
            <Input id="name" v-model="form.name" type="text" autocomplete="name" placeholder="John Doe" />
          </div>
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input id="email" v-model="form.email" type="email" autocomplete="email" placeholder="you@company.com" />
          </div>
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input id="password" v-model="form.password" type="password" autocomplete="new-password" placeholder="Minimal 8 karakter" />
          </div>

          <Button class="w-full" type="submit" :disabled="auth.processing">
            <Spinner v-if="auth.processing" class="mr-2" />
            {{ auth.processing ? 'Creating...' : 'Create Account' }}
          </Button>
        </form>

        <p class="text-muted-foreground text-center text-sm">
          Sudah punya akun?
          <RouterLink to="/login" class="app-auth-link">Sign in</RouterLink>
        </p>
      </CardContent>
    </Card>
  </section>
</template>
