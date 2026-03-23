<template>
  <section class="app-auth-card-grid">
    <div class="app-auth-copy">
      <p class="app-auth-kicker">Gateway Control Plane</p>
      <h1 class="app-auth-title">Masuk ke workspace operasi gateway yang lebih rapi dan terkontrol.</h1>
      <p class="app-auth-description">
        Dashboard ini menyatukan provisioning toko, transaksi, balance, bank, withdraw, dan dokumentasi merchant
        dalam satu permukaan kerja yang konsisten.
      </p>

      <div class="app-auth-points">
        <div class="app-auth-point">
          <strong>Strict Access Control</strong>
          <span>Role, scope toko, dan operasi sensitif tetap divalidasi penuh di backend.</span>
        </div>
        <div class="app-auth-point">
          <strong>Operational Precision</strong>
          <span>Monitor transaksi, saldo, callback, dan settlement tanpa perlu pindah sistem.</span>
        </div>
      </div>
    </div>

    <Card class="app-auth-card">
      <CardHeader>
        <CardTitle>Sign In</CardTitle>
        <CardDescription>Access your dashboard using your registered workspace credentials.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="app-auth-form" @submit.prevent="onSubmit">
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input id="email" v-model="form.email" type="email" autocomplete="email" placeholder="you@company.com" />
          </div>
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input id="password" v-model="form.password" type="password" autocomplete="current-password" placeholder="Masukkan password" />
          </div>
          <p v-if="errorMessage" class="text-sm text-(--danger)">{{ errorMessage }}</p>
          <Button class="w-full" type="submit" :disabled="auth.processing">
            {{ auth.processing ? 'Signing In...' : 'Sign In' }}
          </Button>
        </form>

        <p class="mt-5 text-center text-sm text-muted-foreground">
          Belum punya akun?
          <RouterLink to="/register" class="app-auth-link">Create account</RouterLink>
        </p>
      </CardContent>
    </Card>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const auth = useAuthStore()
const router = useRouter()
const errorMessage = ref('')
const form = reactive({ email: '', password: '' })

const onSubmit = async () => {
  errorMessage.value = ''
  try {
    await auth.login(form)
    await router.push('/dashboard')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Failed to login'
  }
}
</script>
