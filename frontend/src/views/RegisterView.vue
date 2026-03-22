<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { TriangleAlert } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'

const auth = useAuthStore()
const router = useRouter()
const errorMessage = ref('')

const form = reactive({
  name: '',
  email: '',
  password: '',
})

const onSubmit = async () => {
  errorMessage.value = ''
  try {
    await auth.register({
      name: form.name.trim(),
      email: form.email.trim(),
      password: form.password,
    })
    await router.push('/dashboard')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Registrasi gagal'
  }
}
</script>

<template>
  <Card class="app-panel border-none shadow-[0_24px_80px_-52px_rgba(0,0,0,.45)]">
    <CardHeader class="space-y-2 text-center">
      <CardTitle class="text-2xl tracking-tight">Create Account</CardTitle>
      <CardDescription>Daftar untuk mulai menggunakan API gateway workspace.</CardDescription>
    </CardHeader>

    <CardContent class="space-y-5">
      <Alert v-if="errorMessage" variant="destructive">
        <TriangleAlert class="h-4 w-4" />
        <AlertTitle>Registration Error</AlertTitle>
        <AlertDescription>{{ errorMessage }}</AlertDescription>
      </Alert>

      <form class="space-y-4" @submit.prevent="onSubmit">
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
        <RouterLink to="/login" class="text-primary font-medium hover:underline">Sign in</RouterLink>
      </p>
    </CardContent>
  </Card>
</template>
