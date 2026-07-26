export interface Invitation {
  id: string;
  email: string;
  role: 'admin' | 'editor' | 'viewer';
  status: 'pending' | 'accepted' | 'rejected' | 'expired' | 'revoked';
  invitation_token: string;
  expires_at: string;
  project_name?: string;
  project_id?: string;
  created_at: string;
}