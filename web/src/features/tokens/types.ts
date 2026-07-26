export interface ApiToken {
  id: string;
  name: string;
  token?: string;
  last_used_at?: string;
  created_at: string;
  expires_at?: string;
}

export interface CreateTokenInput {
  name: string;
  expires_at?: string;
}