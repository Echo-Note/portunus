import { Modal, Form, Input } from 'antd';
import { useEffect } from 'react';
import { useUpdateProject } from '../hooks';
import type { Project } from '../types';

interface ProjectEditModalProps {
  open: boolean;
  project: Project | null;
  onClose: () => void;
}

interface FormValues {
  name: string;
  description?: string;
}

export function ProjectEditModal({ open, project, onClose }: ProjectEditModalProps) {
  const [form] = Form.useForm<FormValues>();
  const updateProject = useUpdateProject(project?.id || '');

  useEffect(() => {
    if (project && open) {
      form.setFieldsValue({ name: project.name, description: project.description });
    }
  }, [project, open, form]);

  const handleSubmit = async (values: FormValues) => {
    if (!project) return;
    await updateProject.mutateAsync(values);
    form.resetFields();
    onClose();
  };

  const handleCancel = () => { form.resetFields(); onClose(); };

  return (
    <Modal title="编辑项目" open={open} onCancel={handleCancel} onOk={() => form.submit()}
      confirmLoading={updateProject.isPending} width={520} destroyOnHidden>
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item name="name" label="项目名称" rules={[
          { required: true, message: '请输入项目名称' },
          { max: 255, message: '项目名称不超过 255 个字符' },
        ]}>
          <Input placeholder="项目名称" />
        </Form.Item>
        <Form.Item name="description" label="项目描述">
          <Input.TextArea placeholder="项目描述（可选）" rows={3} />
        </Form.Item>
      </Form>
    </Modal>
  );
}