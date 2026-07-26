export interface Member {
  user_id: string;
  project_id: string;
  role: 'owner' | 'admin' | 'editor' | 'viewer';
  status: 'pending' | 'active' | 'removed' | 'left';
  joined_at?: string;
  user?: { id: string; email: string; status: string };
}

export interface InviteMemberInput {
  email: string;
  role: 'admin' | 'editor' | 'viewer';
}

export interface ChangeRoleInput {
  role: 'admin' | 'editor' | 'viewer';
}