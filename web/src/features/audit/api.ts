import apiClient from '../../lib/api-client';
import type { AuditLog } from './types';

export const auditApi = {
  list: (projectId: string, params?: { page?: number; page_size?: number }): Promise<{ items: AuditLog[]; total: number }> =>
    apiClient.get(`/projects/${projectId}/audit-logs`, { params }),
};