import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { membersApi } from './api';
import { InviteMemberInput, ChangeRoleInput } from './types';
import { ApiError } from '../../types/api';

export function useMembers(projectId: string) {
  return useQuery({
    queryKey: ['members', projectId],
    queryFn: () => membersApi.list(projectId),
    staleTime: 30_000,
    enabled: !!projectId,
  });
}

export function useInviteMember(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: InviteMemberInput) => membersApi.invite(projectId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', projectId] });
      message.success('邀请已发送');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '邀请失败');
    },
  });
}

export function useChangeRole(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, input }: { userId: string; input: ChangeRoleInput }) =>
      membersApi.changeRole(projectId, userId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', projectId] });
      message.success('角色已变更');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '变更失败');
    },
  });
}

export function useRemoveMember(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => membersApi.remove(projectId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', projectId] });
      message.success('成员已移除');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '移除失败');
    },
  });
}

export function useLeaveProject(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => membersApi.leave(projectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members', projectId] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('已退出项目');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '操作失败');
    },
  });
}