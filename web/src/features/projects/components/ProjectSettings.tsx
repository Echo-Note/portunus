import { Card, Descriptions, Button, Popconfirm, Space, Alert } from 'antd';
import { Project } from '../types';
import { hasPermission } from '../../../types/rbac';
import { StatusTag } from '../../../components/ui/StatusTag';
import { useDeleteProject, useSuspendProject, useUnsuspendProject } from '../hooks';
import dayjs from 'dayjs';

interface ProjectSettingsProps {
  project: Project;
  currentUserRole?: string;
  onEdit: () => void;
}

/**
 * 项目设置面板：信息展示、冻结/解冻、删除操作。
 */
export function ProjectSettings({ project, currentUserRole, onEdit }: ProjectSettingsProps) {
  const deleteProject = useDeleteProject();
  const suspendProject = useSuspendProject(project.id);
  const unsuspendProject = useUnsuspendProject(project.id);

  const canEdit = hasPermission(currentUserRole, 'projectSettings');
  const canDelete = hasPermission(currentUserRole, 'deleteProject');
  const isActive = project.status === 'active';

  if (project.status === 'deleting') {
    return <Alert type="info" message="项目正在异步删除中，请稍候..." />;
  }

  return (
    <div>
      <Descriptions
        title="项目信息"
        column={2}
        bordered
        extra={
          canEdit && (
            <Button type="primary" onClick={onEdit}>
              编辑
            </Button>
          )
        }
      >
        <Descriptions.Item label="项目名称">{project.name}</Descriptions.Item>
        <Descriptions.Item label="状态">
          <StatusTag status={project.status} />
        </Descriptions.Item>
        <Descriptions.Item label="项目 ID">{project.project_id}</Descriptions.Item>
        <Descriptions.Item label="环境">
          <StatusTag status={project.environment} label={project.environment} />
        </Descriptions.Item>
        <Descriptions.Item label="计划">{project.plan}</Descriptions.Item>
        <Descriptions.Item label="最大域名数">{project.max_domains}</Descriptions.Item>
        <Descriptions.Item label="最大成员数">{project.max_members}</Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {dayjs(project.created_at).format('YYYY-MM-DD HH:mm:ss')}
        </Descriptions.Item>
        <Descriptions.Item label="更新时间">
          {dayjs(project.updated_at).format('YYYY-MM-DD HH:mm:ss')}
        </Descriptions.Item>
      </Descriptions>

      {canEdit && (
        <Card title="危险操作" style={{ marginTop: 24 }} styles={{ header: { color: '#ff4d4f' } }}>
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {isActive ? (
              <Popconfirm
                title={`确定要冻结项目 ${project.name} 吗？`}
                onConfirm={() => suspendProject.mutate()}
              >
                <Button danger loading={suspendProject.isPending}>
                  冻结项目
                </Button>
              </Popconfirm>
            ) : (
              <Popconfirm
                title={`确定要解冻项目 ${project.name} 吗？`}
                onConfirm={() => unsuspendProject.mutate()}
              >
                <Button type="primary" loading={unsuspendProject.isPending}>
                  解冻项目
                </Button>
              </Popconfirm>
            )}

            {canDelete && (
              <Popconfirm
                title={`确定要删除 ${project.name} 吗？此操作不可撤销。`}
                onConfirm={() => deleteProject.mutate(project.id)}
              >
                <Button danger type="primary" loading={deleteProject.isPending}>
                  删除项目
                </Button>
              </Popconfirm>
            )}
          </Space>
        </Card>
      )}
    </div>
  );
}
