import { Select } from 'antd';
import { useChangeRole } from '../hooks';
import type { Member } from '../types';

interface MemberRoleSelectProps { projectId: string; member: Member; }

export function MemberRoleSelect({ projectId, member }: MemberRoleSelectProps) {
  const changeRole = useChangeRole(projectId);

  return (
    <Select size="small" value={member.role} style={{ width: 120 }} loading={changeRole.isPending}
      onChange={(role) => changeRole.mutate({ userId: member.user_id, input: { role: role as 'admin' | 'editor' | 'viewer' } })}
      options={[
        { value: 'admin', label: '管理员' },
        { value: 'editor', label: '编辑者' },
        { value: 'viewer', label: '观察者' },
      ]} />
  );
}