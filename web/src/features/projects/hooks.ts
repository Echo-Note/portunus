import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { projectsApi } from './api';
import { CreateProjectInput, UpdateProjectInput } from './types';
import { ApiError } from '../../types/api';

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: projectsApi.list,
    staleTime: 30_000,
  });
}

export function useProject(projectId: string) {
  return useQuery({
    queryKey: ['projects', projectId],
    queryFn: () => projectsApi.get(projectId),
    enabled: !!projectId,
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateProjectInput) => projectsApi.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('项目创建成功');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '创建失败');
    },
  });
}

export function useUpdateProject(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateProjectInput) => projectsApi.update(projectId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectId] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('项目更新成功');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '更新失败');
    },
  });
}

export function useDeleteProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (projectId: string) => projectsApi.delete(projectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('已提交删除请求，正在异步处理');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '删除失败');
    },
  });
}

export function useSuspendProject(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => projectsApi.suspend(projectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectId] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('项目已冻结');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '操作失败');
    },
  });
}

export function useUnsuspendProject(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => projectsApi.unsuspend(projectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectId] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('项目已解冻');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '操作失败');
    },
  });
}