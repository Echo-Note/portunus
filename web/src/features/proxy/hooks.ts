import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { proxiesApi } from './api';
import { UpdateProxyInput, AddUpstreamInput } from './types';
import { ApiError } from '../../types/api';

export function useProxyConfig(projectId: string, domainId: string) {
  return useQuery({
    queryKey: ['proxy', projectId, domainId],
    queryFn: () => proxiesApi.get(projectId, domainId),
    enabled: !!projectId && !!domainId,
  });
}

export function useUpdateProxy(projectId: string, domainId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateProxyInput) => proxiesApi.update(projectId, domainId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxy', projectId, domainId] });
      message.success('代理配置更新成功');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '更新失败');
    },
  });
}

export function useUpstreamStatus(projectId: string, domainId: string) {
  return useQuery({
    queryKey: ['upstreamStatus', projectId, domainId],
    queryFn: () => proxiesApi.getStatus(projectId, domainId),
    refetchInterval: 15_000, // 每 15 秒轮询
    enabled: !!projectId && !!domainId,
  });
}

export function useAddUpstream(projectId: string, domainId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AddUpstreamInput) => proxiesApi.addUpstream(projectId, domainId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxy', projectId, domainId] });
      queryClient.invalidateQueries({ queryKey: ['upstreamStatus', projectId, domainId] });
      message.success('上游添加成功');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '添加失败');
    },
  });
}

export function useRemoveUpstream(projectId: string, domainId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (upstreamId: string) => proxiesApi.removeUpstream(projectId, domainId, upstreamId),
    onMutate: async (upstreamId) => {
      // 乐观更新：立即从缓存移除
      await queryClient.cancelQueries({ queryKey: ['proxy', projectId, domainId] });
      const previous = queryClient.getQueryData(['proxy', projectId, domainId]);
      return { previous };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxy', projectId, domainId] });
      queryClient.invalidateQueries({ queryKey: ['upstreamStatus', projectId, domainId] });
      message.success('上游已移除');
    },
    onError: (error: ApiError, _vars, ctx) => {
      if (ctx?.previous) {
        queryClient.setQueryData(['proxy', projectId, domainId], ctx.previous);
      }
      message.error(error.message || '移除失败');
    },
  });
}
