import { Card, Button, Spin, Typography, Result, Space, Tag } from 'antd';
import { CheckOutlined, CloseOutlined } from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { message } from 'antd';
import { invitationsApi } from '../../features/invitations/api';
import { ApiError } from '../../types/api';
import type { Invitation } from '../../features/invitations/types';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;

/**
 * 邀请处理页：展示邀请详情，接受/拒绝。
 */
export function InvitationPage() {
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();

  const { data: invitation, isLoading, isError } = useQuery({
    queryKey: ['invitation', token],
    queryFn: () => invitationsApi.get(token!),
    enabled: !!token,
  });

  const acceptMutation = useMutation({
    mutationFn: () => invitationsApi.accept(token!),
    onSuccess: () => {
      message.success('已成功加入项目');
      navigate(`/projects/${invitation?.project_id}`);
    },
    onError: (error: ApiError) => {
      message.error(error.message || '操作失败');
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () => invitationsApi.reject(token!),
    onSuccess: () => {
      message.info('已拒绝邀请');
    },
    onError: (error: ApiError) => {
      message.error(error.message || '操作失败');
    },
  });

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}><Spin size="large" /></div>;
  }

  if (isError || !invitation) {
    return <Result status="error" title="邀请无效" subTitle="此邀请链接无效或已过期" extra={<Button type="primary" onClick={() => navigate('/')}>返回首页</Button>} />;
  }

  if (invitation.status === 'accepted') {
    return <Result status="success" title="邀请已接受" subTitle="您已成功加入此项目" extra={<Button type="primary" onClick={() => navigate(`/projects/${invitation.project_id}`)}>前往项目</Button>} />;
  }

  if (invitation.status === 'rejected') {
    return <Result status="info" title="已拒绝邀请" subTitle="您已拒绝此邀请" extra={<Button onClick={() => navigate('/')}>返回首页</Button>} />;
  }

  if (invitation.status === 'expired') {
    return <Result status="warning" title="邀请已过期" subTitle="此邀请链接已过期，请联系项目管理员重新邀请" extra={<Button onClick={() => navigate('/')}>返回首页</Button>} />;
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', background: '#f5f5f5' }}>
      <Card style={{ width: 480, borderRadius: 12 }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={3}>项目邀请</Title>
        </div>
        <div style={{ marginBottom: 24 }}>
          <Paragraph>
            <Text strong>{invitation.project_name || '未知项目'}</Text> 的管理员邀请您加入项目
          </Paragraph>
          <Space direction="vertical" style={{ width: '100%' }}>
            <div><Text type="secondary">邮箱：</Text><Text>{invitation.email}</Text></div>
            <div><Text type="secondary">角色：</Text><Tag color="blue">{invitation.role}</Tag></div>
            <div><Text type="secondary">过期时间：</Text><Text>{dayjs(invitation.expires_at).format('YYYY-MM-DD HH:mm')}</Text></div>
          </Space>
        </div>
        <Space style={{ width: '100%', justifyContent: 'center' }}>
          <Button type="primary" icon={<CheckOutlined />} onClick={() => acceptMutation.mutate()} loading={acceptMutation.isPending} size="large">接受邀请</Button>
          <Button danger icon={<CloseOutlined />} onClick={() => rejectMutation.mutate()} loading={rejectMutation.isPending} size="large">拒绝</Button>
        </Space>
      </Card>
    </div>
  );
}