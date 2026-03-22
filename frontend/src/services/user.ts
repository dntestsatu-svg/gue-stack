import api from './http'
import type { ApiResponse, User, UserListPage, UserListQuery, UserRole } from './types'

export async function me() {
  const { data } = await api.get<ApiResponse<User>>('/api/v1/user/me')
  return data.data
}

export async function list(query: UserListQuery = {}) {
  const { data } = await api.get<ApiResponse<UserListPage>>('/api/v1/users', {
    params: query,
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

export interface UpdateUserActivePayload {
  is_active: boolean
}

export async function updateActive(userID: number, payload: UpdateUserActivePayload) {
  const { data } = await api.patch<ApiResponse<User>>(`/api/v1/users/${userID}/active`, payload)
  return data.data
}

export async function remove(userID: number) {
  const { data } = await api.delete<ApiResponse<null>>(`/api/v1/users/${userID}`)
  return data.message ?? 'User deleted successfully'
}

export interface ChangePasswordPayload {
  current_password: string
  new_password: string
  confirm_password: string
}

export async function changePassword(payload: ChangePasswordPayload) {
  const { data } = await api.patch<ApiResponse<null>>('/api/v1/user/password', payload)
  return data.message ?? 'Password updated successfully'
}
