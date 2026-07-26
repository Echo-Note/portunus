import { Avatar, Dropdown, MenuProps, Typography } from 'antd';
import { UserOutlined, KeyOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../features/auth/store';
import { useLogout } from '../../features/auth/hooks';

const { Text } = Typography;

/**
 * 用户头像下拉菜单：个人信息、API Token、退出登录。
 */
export function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const logout = useLogout();
  const navigate = useNavigate();

  const items: MenuProps['items'] = [
    {
      key: 'info',
      icon: <UserOutlined />,
      label: user?.email ?? '未登录',
      disabled: true,
    },
    { type: 'divider' },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '个人设置',
      onClick: () => navigate('/settings'),
    },
    {
      key: 'tokens',
      icon: <KeyOutlined />,
      label: 'API Token',
      onClick: () => navigate('/settings'),
    },
    { type: 'divider' },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
      onClick: () => logout.mutate(),
    },
  ];

  return (
    <Dropdown menu={{ items }} placement="topRight" trigger={['click']}>
      <div style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
        <Avatar icon={<UserOutlined />} style={{ backgroundColor: '#1677ff' }} />
        <Text style={{ color: '#fff', maxWidth: 120 }} ellipsis>
          {user?.email ?? ''}
        </Text>
      </div>
    </Dropdown>
  );
}