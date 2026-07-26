import apiClient from '../../lib/api-client';
import { AuthResponse, LoginInput, RegisterInput, User } from './types';

export const authApi = {
  register: (input: RegisterInput): Promise<AuthResponse> =>
    apiClient.post('/auth/register', input),

  login: (input: LoginInput): Promise<AuthResponse> =>
    apiClient.post('/auth/login', input),

  refresh: (refreshToken: string): Promise<{ access_token: string; refresh_token: string }> =>
    apiClient.post('/auth/refresh', { refresh_token: refreshToken }),

  logout: (): Promise<void> =>
    apiClient.post('/auth/logout'),

  getMe: (): Promise<User> =>
    apiClient.get('/me'),

  updateMe: (input: { email?: string }): Promise<User> =>
    apiClient.patch('/me', input),
};