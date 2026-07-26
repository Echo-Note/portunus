import { Card, Descriptions, Spin } from 'antd';
import { useAuthStore } from '../../features/auth/store';
import { PageHeader } from '../../components/ui/PageHeader';
import { StatusTag } from '../../components/ui/StatusTag';
import { TokenTable } from '../../features/tokens/components/TokenTable';
import dayjs from 'dayjs';

/**
 * 个人设置页：用户信息 + API Token 管理。
 */
export function SettingsPage() {
  const user = useAuthStore((s) => s.user);

  if (!user) {
    return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
  }

  return (
    <div>
      <PageHeader title="个人设置" breadcrumbs={[{ title: '仪表盘', path: '/' }, { title: '个人设置' }]} />
      <Card title="个人信息" style={{ marginBottom: 24 }}>
        <Descriptions column={1}>
          <Descriptions.Item label="用户 ID">{user.id}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
          <Descriptions.Item label="状态"><StatusTag status={user.status} /></Descriptions.Item>
          <Descriptions.Item label="注册时间">{dayjs(user.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
        </Descriptions>
      </Card>
      <Card title="API Token">
        <TokenTable />
      </Card>
    </div>
  );
}