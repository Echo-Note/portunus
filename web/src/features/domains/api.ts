import apiClient from '../../lib/api-client';
import type { CreateDomainInput, Domain, UpdateDomainInput } from './types';

export const domainsApi = {
  list: async (projectId: string): Promise<Domain[]> => {
    const data = await apiClient.get(`/projects/${projectId}/domains`) as { items: Domain[]; total: number };
    return data.items ?? [];
  },

  get: (projectId: string, domainId: string): Promise<Domain> =>
    apiClient.get(`/projects/${projectId}/domains/${domainId}`),

  create: (projectId: string, input: CreateDomainInput): Promise<Domain> =>
    apiClient.post(`/projects/${projectId}/domains`, input),

  update: (projectId: string, domainId: string, input: UpdateDomainInput): Promise<Domain> =>
    apiClient.patch(`/projects/${projectId}/domains/${domainId}`, input),

  delete: (projectId: string, domainId: string): Promise<void> =>
    apiClient.delete(`/projects/${projectId}/domains/${domainId}`),
};
