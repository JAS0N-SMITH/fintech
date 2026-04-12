export interface AdminUser {
  id: string;
  email: string;
  display_name: string;
  role: 'user' | 'admin';
  created_at: string;
  updated_at: string;
}

export interface AuditLogEntry {
  id: string;
  user_id: string;
  action: string;
  target_entity: string;
  target_id: string;
  before_value: Record<string, any> | null;
  after_value: Record<string, any> | null;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

export interface AdminUserList {
  users: AdminUser[];
  total: number;
}

export interface AuditLogList {
  entries: AuditLogEntry[];
  total: number;
  page: number;
  page_size: number;
}

export interface AuditLogFilter {
  user_id?: string;
  action?: string;
  from?: string; // RFC3339
  to?: string; // RFC3339
  page: number;
  page_size: number;
}

export interface HealthStatus {
  db: 'healthy' | 'unhealthy' | 'unavailable';
  finnhub_api: 'healthy' | 'unhealthy' | 'unavailable';
  websocket_count: number;
  timestamp: string;
}

export interface PatchRoleInput {
  role: 'user' | 'admin';
}
