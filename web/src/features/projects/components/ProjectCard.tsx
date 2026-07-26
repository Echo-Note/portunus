import { Card, Typography, Tag, Space, Popconfirm, Tooltip } from 'antd';
import {
  ProjectOutlined,
  GlobalOutlined,
  TeamOutlined,
  DeleteOutlined,
  EditOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Project } from '../types';
import { hasPermission } from '../../../types/rbac';
import { StatusTag } from '../../../components/ui/StatusTag';
import { useDeleteProject, useSuspendProject, useUnsuspendProject } from '../hooks';

const { Text, Paragraph } = Typography;

interface ProjectCardProps {
  project: Project;
  currentUserRole?: string;
}

/**
 * 项目卡片：展示名称、描述、域名数、成员数、状态标签及操作按钮。
 */
export function ProjectCard({ project, currentUserRole }: ProjectCardProps) {
  const navigate = useNavigate();
  const deleteProject = useDeleteProject();
  const suspendProject = useSuspendProject(project.id);
  const unsuspendProject = useUnsuspendProject(project.id);

  const canDelete = hasPermission(currentUserRole, 'deleteProject');
  const canEdit = hasPermission(currentUserRole, 'projectSettings');

  const isActive = project.status === 'active';

  return (
    <Card
      hoverable
      onClick={() => navigate(`/projects/${project.id}`)}
      style={{ borderRadius: 8, height: '100%' }}
      actions={[
        canEdit ? (
          <Tooltip title="编辑" key="edit">
            <EditOutlined
              onClick={(e) => {
                e.stopPropagation();
                // 编辑功能由 ProjectEditModal 处理
              }}
            />
          </Tooltip>
        ) : null,
        canEdit ? (
          isActive ? (
            <Tooltip title="冻结" key="suspend">
              <Popconfirm
                title={`确定要冻结项目 ${project.name} 吗？`}
                onConfirm={(e) => {
                  e?.stopPropagation();
                  suspendProject.mutate();
                }}
                onCancel={(e) => e?.stopPropagation()}
              >
                <PauseCircleOutlined
                  onClick={(e) => e.stopPropagation()}
                />
              </Popconfirm>
            </Tooltip>
          ) : (
            <Tooltip title="解冻" key="unsuspend">
              <Popconfirm
                title={`确定要解冻项目 ${project.name} 吗？`}
                onConfirm={(e) => {
                  e?.stopPropagation();
                  unsuspendProject.mutate();
                }}
                onCancel={(e) => e?.stopPropagation()}
              >
                <PlayCircleOutlined
                  onClick={(e) => e.stopPropagation()}
                />
              </Popconfirm>
            </Tooltip>
          )
        ) : null,
        canDelete ? (
          <Tooltip title="删除" key="delete">
            <Popconfirm
              title={`确定要删除 ${project.name} 吗？此操作不可撤销。`}
              onConfirm={(e) => {
                e?.stopPropagation();
                deleteProject.mutate(project.id);
              }}
              onCancel={(e) => e?.stopPropagation()}
            >
              <DeleteOutlined
                style={{ color: '#ff4d4f' }}
                onClick={(e) => e.stopPropagation()}
              />
            </Popconfirm>
          </Tooltip>
        ) : null,
      ].filter(Boolean)}
    >
      <Card.Meta
        title={
          <Space>
            <ProjectOutlined />
            <span>{project.name}</span>
            <StatusTag status={project.status} />
          </Space>
        }
        description={
          <>
            <Paragraph
              type="secondary"
              ellipsis={{ rows: 2 }}
              style={{ marginBottom: 12, minHeight: 44 }}
            >
              {project.description || '暂无描述'}
            </Paragraph>
            <Space split={<span style={{ color: '#d9d9d9' }}>|</span>}>
              <Text type="secondary">
                <GlobalOutlined style={{ marginRight: 4 }} />
                {project.max_domains} 域名
              </Text>
              <Text type="secondary">
                <TeamOutlined style={{ marginRight: 4 }} />
                {project.max_members} 成员
              </Text>
              <Tag>{project.environment}</Tag>
            </Space>
          </>
        }
      />
    </Card>
  );
}
