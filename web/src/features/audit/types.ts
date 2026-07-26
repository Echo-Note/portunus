export interface AuditLog {
  id: number;
  action: string;
  resource_type: string;
  resource_id: string;
  actor_type: 'user' | 'ai_agent' | 'system';
  actor_name: string;
  result: 'success' | 'failed';
  via: 'web_ui' | 'mcp_tool' | 'api_token' | 'system';
  created_at: string;
}