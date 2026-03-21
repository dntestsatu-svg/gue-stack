import { defineStore } from 'pinia'
import * as userApi from '@/services/user'
import type { User } from '@/services/types'

interface UserState {
  profile: User | null
  loading: boolean
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    profile: null,
    loading: false,
  }),
  actions: {
    setProfile(user: User) {
      this.profile = user
    },
    clear() {
      this.profile = null
    },
    async fetchMe() {
      this.loading = true
      try {
        this.profile = await userApi.me()
      } finally {
        this.loading = false
      }
      return this.profile
    },
  },
})
