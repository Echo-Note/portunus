import apiClient from '../../lib/api-client';
import { CreateProjectInput, Project, UpdateProjectInput } from './types';

export const projectsApi = {
  list: (): Promise<Project[]> =>
    apiClient.get('/projects'),

  get: (projectId: string): Promise<Project> =>
    apiClient.get(`/projects/${projectId}`),

  create: (input: CreateProjectInput): Promise<Project> =>
    apiClient.post('/projects', input),

  update: (projectId: string, input: UpdateProjectInput): Promise<Project> =>
    apiClient.patch(`/projects/${projectId}`, input),

  delete: (projectId: string): Promise<void> =>
    apiClient.delete(`/projects/${projectId}`),

  suspend: (projectId: string): Promise<void> =>
    apiClient.post(`/projects/${projectId}/suspend`),

  unsuspend: (projectId: string): Promise<void> =>
    apiClient.post(`/projects/${projectId}/unsuspend`),
};