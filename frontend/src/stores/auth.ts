import { defineStore } from 'pinia'
import * as authApi from '@/services/auth'
import { getApiErrorMessage } from '@/services/http'
import type { AuthResponseData } from '@/services/types'
import { useUserStore } from './user'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  expiresIn: number
  ready: boolean
  processing: boolean
}

const ACCESS_KEY = 'access_token'
const REFRESH_KEY = 'refresh_token'

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    accessToken: null,
    refreshToken: null,
    expiresIn: 0,
    ready: false,
    processing: false,
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.accessToken),
  },
  actions: {
    hydrate() {
      if (this.ready) {
        return
      }
      this.accessToken = localStorage.getItem(ACCESS_KEY)
      this.refreshToken = localStorage.getItem(REFRESH_KEY)
      this.ready = true
    },
    applySession(payload: AuthResponseData) {
      this.accessToken = payload.access_token
      this.refreshToken = payload.refresh_token
      this.expiresIn = payload.expires_in
      localStorage.setItem(ACCESS_KEY, payload.access_token)
      localStorage.setItem(REFRESH_KEY, payload.refresh_token)
      const userStore = useUserStore()
      userStore.setProfile(payload.user)
    },
    clearSession() {
      this.accessToken = null
      this.refreshToken = null
      this.expiresIn = 0
      localStorage.removeItem(ACCESS_KEY)
      localStorage.removeItem(REFRESH_KEY)
      useUserStore().clear()
    },
    async register(payload: authApi.RegisterPayload) {
      this.processing = true
      try {
        const data = await authApi.register(payload)
        this.applySession(data)
        return data
      } catch (error) {
        throw new Error(getApiErrorMessage(error))
      } finally {
        this.processing = false
      }
    },
    async login(payload: authApi.LoginPayload) {
      this.processing = true
      try {
        const data = await authApi.login(payload)
        this.applySession(data)
        return data
      } catch (error) {
        throw new Error(getApiErrorMessage(error))
      } finally {
        this.processing = false
      }
    },
    async tryRefresh() {
      if (!this.refreshToken) {
        return false
      }
      try {
        const data = await authApi.refresh(this.refreshToken)
        this.applySession(data)
        return true
      } catch {
        this.clearSession()
        return false
      }
    },
    async restoreSession() {
      this.hydrate()
      if (this.accessToken) {
        return true
      }
      return this.tryRefresh()
    },
    async logout() {
      if (this.refreshToken) {
        try {
          await authApi.logout(this.refreshToken)
        } catch {
          // ignore remote logout failure; clear local session regardless
        }
      }
      this.clearSession()
    },
  },
})
