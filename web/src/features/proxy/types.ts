export type LBPolicy = 'random' | 'round_robin' | 'least_conn' | 'ip_hash' | 'uri_hash';

export interface ProxyConfig {
  id: string;
  domain_id: string;
  lb_policy: LBPolicy;
  health_check_uri?: string;
  health_check_interval: string;
  timeout: string;
  status: 'active' | 'updating' | 'degraded' | 'unavailable';
}

export interface UpdateProxyInput {
  lb_policy?: LBPolicy;
  health_check_uri?: string;
  health_check_interval?: string;
  timeout?: string;
}

export interface Upstream {
  id: string;
  dial_address: string;
  weight: number;
  status: 'active' | 'unhealthy' | 'disabled' | 'removed';
  proxy_config_id: string;
}

export interface AddUpstreamInput {
  dial_address: string;
  weight?: number;
}

export interface UpstreamStatus {
  upstream_id: string;
  dial_address: string;
  status: string;
  healthy: boolean;
  fails?: number;
}
