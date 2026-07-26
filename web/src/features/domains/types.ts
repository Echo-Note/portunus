export interface Domain {
  id: string;
  domain_name: string;
  status: 'creating' | 'active' | 'updating' | 'error' | 'disabled' | 'deleting' | 'deleted';
  ssl_enabled: boolean;
  caddy_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDomainInput {
  domain_name: string;
  ssl_enabled?: boolean;
}

export interface UpdateDomainInput {
  ssl_enabled?: boolean;
}
