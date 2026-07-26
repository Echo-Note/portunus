import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { domainsApi } from './api';
import { CreateDomainInput, UpdateDomainInput } from './types';
import { ApiError } from '../../types/api';

export function useDomains(projectId: string) {
  return useQuery({
    queryKey: ['domains', projectId],
    queryFn: () => domainsApi.list(projectId),
    staleTime: 30_000,
    enabled: !!projectId,
  });
}

export function useDomain(projectId: string, domainId: string) {
  return useQuery({
    queryKey: ['domains', projectId, domainId],
    queryFn: () => domainsApi.get(projectId, domainId),
    enabled: !!projectId && !!domainId,
  });
}

export function useCreateDomain(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateDomainInput) => domainsApi.create(projectId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['domains', projectId] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('域名创建成功');
    },
    onError: (error: ApiError) => {
      if (error.code === 409) {
        message.error('该域名已存在');
      } else {
        message.error(error.message || '创建失败');
      }
    },
  });
}

export function useUpdateDomain(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ domainId, input }: { domainId: string; input: UpdateDomainInput }) =>
      domainsApi.update(projectId, domainId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['domains', projectId] });
      message.success('域名更新成功');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '更新失败');
    },
  });
}

export function useDeleteDomain(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (domainId: string) => domainsApi.delete(projectId, domainId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['domains', projectId] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('域名删除成功');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '删除失败');
    },
  });
}
