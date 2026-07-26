/** 状态颜色映射 */
export const STATUS_COLORS: Record<string, string> = {
  active: 'green',
  pending: 'orange',
  suspended: 'red',
  deleted: 'default',
  deleting: 'processing',
  error: 'red',
  creating: 'processing',
  updating: 'processing',
  disabled: 'warning',
  unhealthy: 'red',
  healthy: 'green',
  unavailable: 'red',
  degraded: 'orange',
  removed: 'default',
  left: 'default',
  accepted: 'green',
  rejected: 'red',
  expired: 'default',
  revoked: 'default',
  success: 'green',
  failed: 'red',
};

/** 角色权限矩阵 */
export const ROLE_PERMISSIONS = {
  createDomain: ['owner', 'admin', 'editor'],
  editDomain: ['owner', 'admin', 'editor'],
  deleteDomain: ['owner', 'admin'],
  manageUpstream: ['owner', 'admin', 'editor'],
  inviteMember: ['owner', 'admin'],
  removeMember: ['owner', 'admin'],
  changeRole: ['owner', 'admin'],
  projectSettings: ['owner'],
  deleteProject: ['owner'],
  viewAudit: ['owner', 'admin'],
} as const;

/** 检查角色是否有权限执行某操作 */
export function hasPermission(role: string | undefined, action: keyof typeof ROLE_PERMISSIONS): boolean {
  if (!role) return false;
  return (ROLE_PERMISSIONS[action] as readonly string[]).includes(role);
}