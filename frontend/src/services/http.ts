import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  timeout: 10000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export function getApiErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const serverMessage = (error.response?.data as { message?: string } | undefined)?.message
    return serverMessage ?? error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}

export default api
