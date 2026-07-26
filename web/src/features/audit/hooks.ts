import { useQuery } from '@tanstack/react-query';
import { auditApi } from './api';

export function useAuditLogs(projectId: string, page: number = 1, pageSize: number = 20) {
  return useQuery({
    queryKey: ['auditLogs', projectId, page, pageSize],
    queryFn: () => auditApi.list(projectId, { page, page_size: pageSize }),
    enabled: !!projectId,
  });
}