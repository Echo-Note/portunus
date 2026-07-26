import { Modal, Form, Input, Select } from 'antd';
import { useInviteMember } from '../hooks';

interface MemberInviteModalProps { open: boolean; projectId: string; onClose: () => void; }

export function MemberInviteModal({ open, projectId, onClose }: MemberInviteModalProps) {
  const inviteMember = useInviteMember(projectId);
  const [form] = Form.useForm<{ email: string; role: 'admin' | 'editor' | 'viewer' }>();

  const handleSubmit = async (values: { email: string; role: 'admin' | 'editor' | 'viewer' }) => {
    await inviteMember.mutateAsync(values);
    form.resetFields();
    onClose();
  };

  return (
    <Modal title="邀请成员" open={open} onCancel={onClose} onOk={() => form.submit()}
      confirmLoading={inviteMember.isPending} width={520} destroyOnHidden>
      <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ role: 'viewer' }}>
        <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入有效的邮箱地址' }]}>
          <Input placeholder="user@example.com" />
        </Form.Item>
        <Form.Item name="role" label="角色">
          <Select options={[
            { value: 'admin', label: '管理员 - 可管理域名、上游、成员' },
            { value: 'editor', label: '编辑者 - 可管理域名、上游' },
            { value: 'viewer', label: '观察者 - 只读权限' },
          ]} />
        </Form.Item>
      </Form>
    </Modal>
  );
}