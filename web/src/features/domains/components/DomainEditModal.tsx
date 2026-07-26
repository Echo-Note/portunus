import { Modal, Form, Switch } from 'antd';
import { useEffect } from 'react';
import { useUpdateDomain } from '../hooks';
import { Domain } from '../types';

interface DomainEditModalProps {
  open: boolean;
  projectId: string;
  domain: Domain | null;
  onClose: () => void;
}

interface FormValues {
  ssl_enabled: boolean;
}

/**
 * 编辑域名弹窗。
 */
export function DomainEditModal({ open, projectId, domain, onClose }: DomainEditModalProps) {
  const updateDomain = useUpdateDomain(projectId);
  const [form] = Form.useForm<FormValues>();

  useEffect(() => {
    if (domain && open) {
      form.setFieldsValue({ ssl_enabled: domain.ssl_enabled });
    }
  }, [domain, open, form]);

  const handleSubmit = async (values: FormValues) => {
    if (!domain) return;
    await updateDomain.mutateAsync({ domainId: domain.id, input: values });
    form.resetFields();
    onClose();
  };

  return (
    <Modal
      title={`编辑域名: ${domain?.domain_name}`}
      open={open}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={updateDomain.isPending}
      width={520}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item name="ssl_enabled" label="SSL" valuePropName="checked">
          <Switch checkedChildren="启用" unCheckedChildren="禁用" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
