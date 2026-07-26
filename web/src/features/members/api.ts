import apiClient from '../../lib/api-client';
import { ChangeRoleInput, InviteMemberInput, Member } from './types';

export const membersApi = {
  list: (projectId: string): Promise<Member[]> =>
    apiClient.get(`/projects/${projectId}/members`),

  invite: (projectId: string, input: InviteMemberInput): Promise<Member> =>
    apiClient.post(`/projects/${projectId}/members`, input),

  changeRole: (projectId: string, userId: string, input: ChangeRoleInput): Promise<Member> =>
    apiClient.patch(`/projects/${projectId}/members/${userId}`, input),

  remove: (projectId: string, userId: string): Promise<void> =>
    apiClient.delete(`/projects/${projectId}/members/${userId}`),

  leave: (projectId: string): Promise<void> =>
    apiClient.post(`/projects/${projectId}/members/me/leave`),
};