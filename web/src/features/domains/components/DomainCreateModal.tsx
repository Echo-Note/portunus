import { Modal, Form, Input, Switch } from 'antd';
import { useCreateDomain } from '../hooks';

interface DomainCreateModalProps {
  open: boolean;
  projectId: string;
  onClose: () => void;
}

interface FormValues {
  domain_name: string;
  ssl_enabled: boolean;
}

/**
 * 创建域名弹窗。
 */
export function DomainCreateModal({ open, projectId, onClose }: DomainCreateModalProps) {
  const createDomain = useCreateDomain(projectId);
  const [form] = Form.useForm<FormValues>();

  const handleSubmit = async (values: FormValues) => {
    await createDomain.mutateAsync(values);
    form.resetFields();
    onClose();
  };

  return (
    <Modal
      title="创建域名"
      open={open}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={createDomain.isPending}
      width={520}
      destroyOnHidden
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{ ssl_enabled: true }}
      >
        <Form.Item
          name="domain_name"
          label="域名"
          rules={[
            { required: true, message: '请输入域名' },
            { pattern: /^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/, message: '请输入有效的域名格式' },
          ]}
        >
          <Input placeholder="example.com" />
        </Form.Item>

        <Form.Item name="ssl_enabled" label="SSL" valuePropName="checked">
          <Switch checkedChildren="启用" unCheckedChildren="禁用" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
