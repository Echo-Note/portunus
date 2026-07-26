import { Tabs, Spin, Button, Space } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { useParams } from 'react-router-dom';
import { useState } from 'react';
import { useProject } from '../../features/projects/hooks';
import { useMembers } from '../../features/members/hooks';
import { useAuthStore } from '../../features/auth/store';
import { hasPermission } from '../../types/rbac';
import { StatusTag } from '../../components/ui/StatusTag';
import { PageHeader } from '../../components/ui/PageHeader';
import { ErrorResult } from '../../components/ui/ErrorResult';
import { DomainTable } from '../../features/domains/components/DomainTable';
import { DomainCreateModal } from '../../features/domains/components/DomainCreateModal';
import { MemberTable } from '../../features/members/components/MemberTable';
import { ProjectSettings } from '../../features/projects/components/ProjectSettings';
import { ProjectEditModal } from '../../features/projects/components/ProjectEditModal';
import { AuditLogTable } from '../../features/audit/components/AuditLogTable';

/**
 * 项目详情页：Tab 切换域名/成员/审计/设置。
 */
export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const { data: project, isLoading, isError, refetch } = useProject(projectId!);
  const { data: members = [] } = useMembers(projectId!);
  const currentUser = useAuthStore((s) => s.user);
  const [createDomainOpen, setCreateDomainOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);

  const currentMember = members.find((m) => m.user_id === currentUser?.id);
  const currentRole = currentMember?.role;

  if (isLoading) {
    return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
  }

  if (isError || !project) {
    return <ErrorResult message="无法加载项目信息" onRetry={() => refetch()} />;
  }

  const canViewAudit = hasPermission(currentRole, 'viewAudit');
  const canEdit = hasPermission(currentRole, 'projectSettings');
  const canCreateDomain = hasPermission(currentRole, 'createDomain');

  const tabItems = [
    {
      key: 'domains',
      label: '域名',
      children: (
        <div>
          <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
            {canCreateDomain && (
              <Button type="primary" onClick={() => setCreateDomainOpen(true)}>创建域名</Button>
            )}
          </div>
          <DomainTable projectId={projectId!} currentUserRole={currentRole} />
        </div>
      ),
    },
    {
      key: 'members',
      label: '成员',
      children: <MemberTable projectId={projectId!} currentUserRole={currentRole} />,
    },
    ...(canViewAudit ? [{
      key: 'audit',
      label: '审计日志',
      children: <AuditLogTable projectId={projectId!} />,
    }] : []),
    {
      key: 'settings',
      label: '设置',
      children: <ProjectSettings project={project} currentUserRole={currentRole} onEdit={() => setEditModalOpen(true)} />,
    },
  ];

  return (
    <div>
      <PageHeader
        title={<Space>{project.name}<StatusTag status={project.status} /></Space>}
        breadcrumbs={[{ title: '仪表盘', path: '/' }, { title: project.name }]}
        extra={canEdit && <Button icon={<EditOutlined />} onClick={() => setEditModalOpen(true)}>编辑</Button>}
      />

      <Tabs defaultActiveKey="domains" items={tabItems} />

      <DomainCreateModal open={createDomainOpen} projectId={projectId!} onClose={() => setCreateDomainOpen(false)} />
      <ProjectEditModal open={editModalOpen} project={project} onClose={() => setEditModalOpen(false)} />
    </div>
  );
}