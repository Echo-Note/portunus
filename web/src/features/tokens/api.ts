import apiClient from '../../lib/api-client';
import { ApiToken, CreateTokenInput } from './types';

export const tokensApi = {
  list: (): Promise<ApiToken[]> => apiClient.get('/me/tokens'),
  create: (input: CreateTokenInput): Promise<ApiToken> => apiClient.post('/me/tokens', input),
  revoke: (tokenId: string): Promise<void> => apiClient.delete(`/me/tokens/${tokenId}`),
};