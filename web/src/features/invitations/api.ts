import apiClient from '../../lib/api-client';
import type { Invitation } from './types';

export const invitationsApi = {
  get: (token: string): Promise<Invitation> => apiClient.get(`/invitations/${token}`),
  accept: (token: string): Promise<void> => apiClient.post(`/invitations/${token}/accept`),
  reject: (token: string): Promise<void> => apiClient.post(`/invitations/${token}/reject`),
};