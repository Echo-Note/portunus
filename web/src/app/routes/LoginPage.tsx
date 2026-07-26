import { Typography, Card } from 'antd';
import { LoginForm } from '../../features/auth/components/LoginForm';

const { Title, Text } = Typography;

/**
 * 登录页：居中卡片布局，左侧品牌，右侧表单。
 */
export function LoginPage() {
  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        background: 'linear-gradient(135deg, #1677ff 0%, #0958d9 100%)',
      }}
    >
      <Card style={{ width: 420, borderRadius: 12 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={2} style={{ marginBottom: 4 }}>
            Portunus
          </Title>
          <Text type="secondary">反向代理管理平台</Text>
        </div>
        <LoginForm />
      </Card>
    </div>
  );
}