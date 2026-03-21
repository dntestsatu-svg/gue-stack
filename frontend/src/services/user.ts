import api from './http'
import type { ApiResponse, User } from './types'

export async function me() {
  const { data } = await api.get<ApiResponse<User>>('/api/v1/user/me')
  return data.data
}
