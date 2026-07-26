import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { tokensApi } from './api';
import { CreateTokenInput } from './types';
import { ApiError } from '../../types/api';

export function useTokens() {
  return useQuery({ queryKey: ['tokens'], queryFn: tokensApi.list, staleTime: 30_000 });
}

export function useCreateToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTokenInput) => tokensApi.create(input),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['tokens'] }); message.success('Token 创建成功'); },
    onError: (error: ApiError) => { message.error(error.message || '创建失败'); },
  });
}

export function useRevokeToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tokenId: string) => tokensApi.revoke(tokenId),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['tokens'] }); message.success('Token 已撤销'); },
    onError: (error: ApiError) => { message.error(error.message || '撤销失败'); },
  });
}