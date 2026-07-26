import { Form, Input, Button, Progress } from 'antd';
import { MailOutlined, LockOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { useState } from 'react';
import { useRegister } from '../hooks';

interface RegisterFormValues {
  email: string;
  password: string;
  confirmPassword: string;
}

/**
 * 注册表单：邮箱 + 密码 + 确认密码，密码强度校验。
 */
export function RegisterForm() {
  const register = useRegister();
  const [form] = Form.useForm<RegisterFormValues>();
  const [password, setPassword] = useState('');

  const getPasswordStrength = (pwd: string): { percent: number; status: 'exception' | 'active' | 'success'; text: string } => {
    if (!pwd) return { percent: 0, status: 'exception', text: '' };
    let score = 0;
    if (pwd.length >= 8) score += 25;
    if (pwd.length >= 12) score += 25;
    if (/[a-z]/.test(pwd) && /[A-Z]/.test(pwd)) score += 15;
    if (/\d/.test(pwd)) score += 15;
    if (/[^a-zA-Z\d]/.test(pwd)) score += 20;

    if (score < 40) return { percent: score, status: 'exception', text: '弱' };
    if (score < 70) return { percent: score, status: 'active', text: '中等' };
    return { percent: score, status: 'success', text: '强' };
  };

  const strength = getPasswordStrength(password);

  const handleSubmit = (values: RegisterFormValues) => {
    register.mutate({ email: values.email, password: values.password });
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
        rules={[
          { required: true, message: '请输入密码' },
          { min: 8, message: '密码至少 8 位' },
        ]}
      >
        <Input.Password
          prefix={<LockOutlined />}
          placeholder="密码（至少 8 位）"
          autoComplete="new-password"
          onChange={(e) => setPassword(e.target.value)}
        />
      </Form.Item>

      {password && (
        <div style={{ marginBottom: 16, marginTop: -8 }}>
          <Progress
            percent={strength.percent}
            status={strength.status}
            showInfo={false}
            size="small"
          />
          <span style={{ fontSize: 12, color: '#888' }}>密码强度：{strength.text}</span>
        </div>
      )}

      <Form.Item
        name="confirmPassword"
        dependencies={['password']}
        rules={[
          { required: true, message: '请确认密码' },
          ({ getFieldValue }) => ({
            validator(_, value) {
              if (!value || getFieldValue('password') === value) {
                return Promise.resolve();
              }
              return Promise.reject(new Error('两次输入的密码不一致'));
            },
          }),
        ]}
      >
        <Input.Password
          prefix={<LockOutlined />}
          placeholder="确认密码"
          autoComplete="new-password"
        />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit" block loading={register.isPending}>
          注册
        </Button>
      </Form.Item>

      <div style={{ textAlign: 'center' }}>
        <Link to="/login">已有账号？登录</Link>
      </div>
    </Form>
  );
}