import { Button, Space, Popconfirm, Tag, Tooltip } from 'antd';
import { UserAddOutlined, DeleteOutlined, UserDeleteOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useMembers, useRemoveMember, useLeaveProject } from '../hooks';
import type { Member } from '../types';
import { hasPermission } from '../../../types/rbac';
import { DataTable } from '../../../components/ui/DataTable';
import { MemberInviteModal } from './MemberInviteModal';
import { MemberRoleSelect } from './MemberRoleSelect';
import type { ColumnsType } from 'antd/es/table';

interface MemberTableProps {
  projectId: string;
  currentUserRole?: string;
}

const ROLE_COLORS: Record<string, string> = { owner: 'gold', admin: 'blue', editor: 'green', viewer: 'default' };
const ROLE_LABELS: Record<string, string> = { owner: '拥有者', admin: '管理员', editor: '编辑者', viewer: '观察者' };

export function MemberTable({ projectId, currentUserRole }: MemberTableProps) {
  const { data: members = [], isLoading, isError, refetch } = useMembers(projectId);
  const removeMember = useRemoveMember(projectId);
  const leaveProject = useLeaveProject(projectId);
  const [inviteModalOpen, setInviteModalOpen] = useState(false);

  const canInvite = hasPermission(currentUserRole, 'inviteMember');
  const canRemove = hasPermission(currentUserRole, 'removeMember');
  const canChangeRole = hasPermission(currentUserRole, 'changeRole');

  const columns: ColumnsType<Member> = [
    { title: '用户', key: 'user', render: (_: unknown, r: Member) => <span>{r.user?.email || r.user_id}</span> },
    { title: '角色', dataIndex: 'role', key: 'role', width: 180,
      render: (role: string, record: Member) => {
        if (role === 'owner' || !canChangeRole) return <Tag color={ROLE_COLORS[role]}>{ROLE_LABELS[role]}</Tag>;
        return <MemberRoleSelect projectId={projectId} member={record} />;
      },
    },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (status: string) => {
        const colors: Record<string, string> = { active: 'green', pending: 'orange', removed: 'red', left: 'default' };
        return <Tag color={colors[status] || 'default'}>{status}</Tag>;
      },
    },
    ...(canRemove ? [{ title: '操作', key: 'actions', width: 120,
      render: (_: unknown, record: Member) => {
        if (record.role === 'owner') return null;
        return (
          <Popconfirm title="确定要移除该成员吗？此操作不可撤销。" onConfirm={() => removeMember.mutate(record.user_id)}>
            <Tooltip title="移除"><Button type="link" size="small" danger icon={<DeleteOutlined />} loading={removeMember.isPending} /></Tooltip>
          </Popconfirm>
        );
      },
    }] : []),
  ];

  return (
    <>
      <DataTable
        columns={columns}
        dataSource={members}
        rowKey="user_id"
        loading={isLoading}
        error={isError}
        onRetry={() => refetch()}
        emptyText="暂无成员"
        pagination={false}
        toolbar={
          <>
            <strong>成员列表</strong>
            <Space>
              {canInvite && <Button type="primary" icon={<UserAddOutlined />} onClick={() => setInviteModalOpen(true)}>邀请成员</Button>}
              {currentUserRole && currentUserRole !== 'owner' && (
                <Popconfirm title="确定要退出此项目吗？" onConfirm={() => leaveProject.mutate()}>
                  <Button icon={<UserDeleteOutlined />} loading={leaveProject.isPending}>退出项目</Button>
                </Popconfirm>
              )}
            </Space>
          </>
        }
      />
      <MemberInviteModal open={inviteModalOpen} projectId={projectId} onClose={() => setInviteModalOpen(false)} />
    </>
  );
}