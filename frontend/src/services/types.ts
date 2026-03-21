export interface ApiResponse<T> {
  status: string
  data: T
  message?: string
}

export interface User {
  id: number
  name: string
  email: string
}

export interface AuthResponseData {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}
