import api from './http'
import type { ApiResponse, User, UserRole } from './types'

export async function me() {
  const { data } = await api.get<ApiResponse<User>>('/api/v1/user/me')
  return data.data
}

export async function list(limit = 50) {
  const { data } = await api.get<ApiResponse<User[]>>('/api/v1/users', {
    params: { limit },
  })
  return data.data
}

export interface CreateUserPayload {
  name: string
  email: string
  password: string
  role?: UserRole
  is_active?: boolean
}

export async function create(payload: CreateUserPayload) {
  const { data } = await api.post<ApiResponse<User>>('/api/v1/users', payload)
  return data.data
}

export interface UpdateUserRolePayload {
  role: UserRole
}

export async function updateRole(userID: number, payload: UpdateUserRolePayload) {
  const { data } = await api.patch<ApiResponse<User>>(`/api/v1/users/${userID}/role`, payload)
  return data.data
}
