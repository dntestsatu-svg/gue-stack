<template>
  <section class="space-y-6">
    <Card>
      <CardHeader>
        <CardTitle>Dashboard</CardTitle>
        <CardDescription>Protected page accessible only for authenticated users.</CardDescription>
      </CardHeader>
      <CardContent>
        <div v-if="user.profile" class="space-y-2">
          <p><span class="font-semibold">Name:</span> {{ user.profile.name }}</p>
          <p><span class="font-semibold">Email:</span> {{ user.profile.email }}</p>
          <p><span class="font-semibold">User ID:</span> {{ user.profile.id }}</p>
        </div>
        <p v-else class="text-slate-500">Loading profile...</p>
      </CardContent>
      <CardFooter>
        <Button variant="outline" @click="handleLogout">Logout</Button>
      </CardFooter>
    </Card>
  </section>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

const auth = useAuthStore()
const user = useUserStore()
const router = useRouter()

onMounted(async () => {
  if (!user.profile) {
    try {
      await user.fetchMe()
    } catch {
      const refreshed = await auth.tryRefresh()
      if (refreshed) {
        await user.fetchMe()
      } else {
        await router.push('/login')
      }
    }
  }
})

const handleLogout = async () => {
  await auth.logout()
  await router.push('/login')
}
</script>
