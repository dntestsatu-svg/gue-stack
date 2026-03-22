import { defineStore } from 'pinia'
import * as authApi from '@/services/auth'
import { clearCSRFToken, getApiErrorMessage } from '@/services/http'
import type { AuthResponseData } from '@/services/types'
import { useUserStore } from './user'

interface AuthState {
  expiresIn: number
  ready: boolean
  processing: boolean
  authenticated: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    expiresIn: 0,
    ready: false,
    processing: false,
    authenticated: false,
  }),
  getters: {
    isAuthenticated: (state) => state.authenticated,
  },
  actions: {
    applySession(payload: AuthResponseData) {
      this.expiresIn = payload.expires_in
      this.authenticated = true
      const userStore = useUserStore()
      userStore.setProfile(payload.user)
    },
    clearSession() {
      this.expiresIn = 0
      this.authenticated = false
      clearCSRFToken()
      useUserStore().clear()
    },
    async login(payload: authApi.LoginPayload) {
      this.processing = true
      try {
        await authApi.initCsrf()
        const data = await authApi.login(payload)
        this.applySession(data)
        this.ready = true
        return data
      } catch (error) {
        throw new Error(getApiErrorMessage(error))
      } finally {
        this.processing = false
      }
    },
    async register(payload: authApi.RegisterPayload) {
      this.processing = true
      try {
        await authApi.initCsrf()
        const data = await authApi.register(payload)
        this.applySession(data)
        this.ready = true
        return data
      } catch (error) {
        throw new Error(getApiErrorMessage(error))
      } finally {
        this.processing = false
      }
    },
    async tryRefresh() {
      if (!authApi.hasSessionHint()) {
        this.clearSession()
        return false
      }
      try {
        await authApi.initCsrf()
        const data = await authApi.refresh()
        this.applySession(data)
        return true
      } catch {
        this.clearSession()
        return false
      }
    },
    async restoreSession() {
      if (this.ready) {
        return this.authenticated
      }

      if (!authApi.hasSessionHint()) {
        this.clearSession()
        this.ready = true
        return false
      }

      const userStore = useUserStore()
      try {
        await userStore.fetchMe()
        this.authenticated = true
        this.ready = true
        return true
      } catch {
        // try refresh fallback below
      }

      const refreshed = await this.tryRefresh()
      if (!refreshed) {
        this.ready = true
        return false
      }

      try {
        await userStore.fetchMe()
        this.authenticated = true
        this.ready = true
        return true
      } catch {
        this.clearSession()
        this.ready = true
        return false
      }
    },
    async logout() {
      try {
        await authApi.initCsrf()
        await authApi.logout()
      } catch {
        // ignore remote logout failure; clear local session regardless
      }
      this.clearSession()
    },
  },
})
