import apiClient from '../../lib/api-client';
import { CreateDomainInput, Domain, UpdateDomainInput } from './types';

export const domainsApi = {
  list: (projectId: string): Promise<Domain[]> =>
    apiClient.get(`/projects/${projectId}/domains`),

  get: (projectId: string, domainId: string): Promise<Domain> =>
    apiClient.get(`/projects/${projectId}/domains/${domainId}`),

  create: (projectId: string, input: CreateDomainInput): Promise<Domain> =>
    apiClient.post(`/projects/${projectId}/domains`, input),

  update: (projectId: string, domainId: string, input: UpdateDomainInput): Promise<Domain> =>
    apiClient.patch(`/projects/${projectId}/domains/${domainId}`, input),

  delete: (projectId: string, domainId: string): Promise<void> =>
    apiClient.delete(`/projects/${projectId}/domains/${domainId}`),
};
