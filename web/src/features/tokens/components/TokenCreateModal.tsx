import { Modal, Form, Input, DatePicker, Alert, Typography } from 'antd';
import { useState } from 'react';
import { useCreateToken } from '../hooks';
import type { ApiToken } from '../types';

const { Text, Paragraph } = Typography;

interface TokenCreateModalProps { open: boolean; onClose: () => void; }

export function TokenCreateModal({ open, onClose }: TokenCreateModalProps) {
  const createToken = useCreateToken();
  const [form] = Form.useForm<{ name: string; expires_at?: string }>();
  const [newToken, setNewToken] = useState<ApiToken | null>(null);

  const handleSubmit = async (values: { name: string; expires_at?: string }) => {
    const result = await createToken.mutateAsync(values);
    setNewToken(result);
    form.resetFields();
  };

  const handleClose = () => { setNewToken(null); form.resetFields(); onClose(); };

  return (
    <Modal title="创建 API Token" open={open} onCancel={handleClose}
      onOk={newToken ? handleClose : () => form.submit()} confirmLoading={createToken.isPending}
      okText={newToken ? '关闭' : '创建'} width={520} destroyOnHidden>
      {newToken?.token ? (
        <div>
          <Alert type="warning" message="请立即复制此 Token，关闭后将无法再次查看" style={{ marginBottom: 16 }} />
          <Paragraph copyable style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, wordBreak: 'break-all' }}>
            <Text code>{newToken.token}</Text>
          </Paragraph>
        </div>
      ) : (
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="Token 名称" rules={[{ required: true, message: '请输入 Token 名称' }]}>
            <Input placeholder="我的 Token" />
          </Form.Item>
          <Form.Item name="expires_at" label="过期时间（可选）">
            <DatePicker style={{ width: '100%' }} showTime />
          </Form.Item>
        </Form>
      )}
    </Modal>
  );
}