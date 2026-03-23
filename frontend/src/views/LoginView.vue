<template>
  <section class="mx-auto max-w-md">
    <Card>
      <CardHeader>
        <CardTitle>Login</CardTitle>
        <CardDescription>Access your dashboard using your credentials.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="onSubmit">
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input id="email" v-model="form.email" type="email" placeholder="you@example.com" />
          </div>
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input id="password" v-model="form.password" type="password" placeholder="********" />
          </div>
          <p v-if="errorMessage" class="text-sm text-(--danger)">{{ errorMessage }}</p>
          <Button class="w-full" type="submit" :disabled="auth.processing">
            {{ auth.processing ? 'Signing In...' : 'Sign In' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
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
