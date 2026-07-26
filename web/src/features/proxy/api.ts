import apiClient from '../../lib/api-client';
import { AddUpstreamInput, ProxyConfig, UpdateProxyInput, Upstream, UpstreamStatus } from './types';

export const proxiesApi = {
  get: (projectId: string, domainId: string): Promise<ProxyConfig> =>
    apiClient.get(`/projects/${projectId}/domains/${domainId}/proxy`),

  update: (projectId: string, domainId: string, input: UpdateProxyInput): Promise<ProxyConfig> =>
    apiClient.patch(`/projects/${projectId}/domains/${domainId}/proxy`, input),

  addUpstream: (projectId: string, domainId: string, input: AddUpstreamInput): Promise<Upstream> =>
    apiClient.post(`/projects/${projectId}/domains/${domainId}/proxy/upstreams`, input),

  removeUpstream: (projectId: string, domainId: string, upstreamId: string): Promise<void> =>
    apiClient.delete(`/projects/${projectId}/domains/${domainId}/proxy/upstreams/${upstreamId}`),

  getStatus: (projectId: string, domainId: string): Promise<UpstreamStatus[]> =>
    apiClient.get(`/projects/${projectId}/domains/${domainId}/status`),
};
