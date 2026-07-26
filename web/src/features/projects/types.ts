export interface Project {
  id: string;
  project_id: string;
  name: string;
  description?: string;
  status: 'active' | 'suspended' | 'deleting' | 'error' | 'deleted';
  plan: 'free' | 'pro' | 'enterprise';
  environment: 'development' | 'staging' | 'production';
  max_domains: number;
  max_members: number;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectInput {
  project_id: string;
  name: string;
  description?: string;
}

export interface UpdateProjectInput {
  name?: string;
  description?: string;
}