import { Form, Input, Button } from 'antd';
import { MailOutlined, LockOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { useLogin } from '../hooks';

interface LoginFormValues {
  email: string;
  password: string;
}

/**
 * 登录表单：邮箱 + 密码，支持 Enter 提交。
 */
export function LoginForm() {
  const login = useLogin();
  const [form] = Form.useForm<LoginFormValues>();

  const handleSubmit = (values: LoginFormValues) => {
    login.mutate(values);
  };

  return (
    <Form form={form} onFinish={handleSubmit} layout="vertical" size="large">
      <Form.Item
        name="email"
        rules={[
          { required: true, message: '请输入邮箱' },
          { type: 'email', message: '请输入有效的邮箱地址' },
        ]}
      >
        <Input prefix={<MailOutlined />} placeholder="邮箱" autoComplete="email" />
      </Form.Item>

      <Form.Item
        name="password"
        rules={[{ required: true, message: '请输入密码' }]}
      >
        <Input.Password
          prefix={<LockOutlined />}
          placeholder="密码"
          autoComplete="current-password"
        />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit" block loading={login.isPending}>
          登录
        </Button>
      </Form.Item>

      <div style={{ textAlign: 'center' }}>
        <Link to="/register">还没有账号？注册</Link>
      </div>
    </Form>
  );
}