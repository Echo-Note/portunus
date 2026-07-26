import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { message } from 'antd';
import { authApi } from './api';
import { useAuthStore } from './store';
import { ApiError } from '../../types/api';

export function useLogin() {
  const navigate = useNavigate();
  const loginToStore = useAuthStore((s) => s.login);

  return useMutation({
    mutationFn: authApi.login,
    onSuccess: (data) => {
      loginToStore(data.user, data.access_token, data.refresh_token);
      message.success('登录成功');
      const params = new URLSearchParams(window.location.search);
      const redirect = params.get('redirect') || '/';
      navigate(redirect, { replace: true });
    },
    onError: (error: ApiError) => {
      if (error.code === 401) {
        message.error('邮箱或密码错误');
      } else {
        message.error(error.message || '登录失败');
      }
    },
  });
}

export function useRegister() {
  const navigate = useNavigate();
  const registerToStore = useAuthStore((s) => s.register);

  return useMutation({
    mutationFn: authApi.register,
    onSuccess: (data) => {
      registerToStore(data.user, data.access_token, data.refresh_token);
      message.success('注册成功，请查收验证邮件激活账号');
      navigate('/login', { replace: true });
    },
    onError: (error: ApiError) => {
      if (error.code === 409) {
        message.error('该邮箱已注册');
      } else {
        message.error(error.message || '注册失败');
      }
    },
  });
}

export function useLogout() {
  const navigate = useNavigate();
  const logoutFromStore = useAuthStore((s) => s.logout);

  return useMutation({
    mutationFn: authApi.logout,
    onSettled: () => {
      logoutFromStore();
      navigate('/login', { replace: true });
    },
  });
}