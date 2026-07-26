import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { message } from 'antd';
import { ApiError } from '../types/api';
import { useAuthStore } from '../features/auth/store';

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 15_000,
  headers: { 'Content-Type': 'application/json' },
});

// 是否正在刷新 token，防止并发刷新
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: Error) => void;
}> = [];

function processQueue(error: Error | null, token: string | null) {
  failedQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else if (token) {
      resolve(token);
    }
  });
  failedQueue = [];
}

// ── 请求拦截器：自动附加 Bearer token ──
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const { accessToken } = useAuthStore.getState();
  if (accessToken && config.headers) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

// ── 响应拦截器：统一解包 + 401 自动刷新 + 错误处理 ──
apiClient.interceptors.response.use(
  (response) => {
    const body = response.data;
    if (body && typeof body.code === 'number') {
      if (body.code === 0) {
        return body.data;
      }
      throw new ApiError(body.code, body.message || '未知错误');
    }
    return body;
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    if (!error.response) {
      message.error('网络连接失败');
      return Promise.reject(new ApiError(0, '网络连接失败'));
    }

    const { status } = error.response;
    const body = error.response.data as { code?: number; message?: string } | undefined;

    // 401 — 尝试刷新 token
    if (status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({
            resolve: (token: string) => {
              if (originalRequest.headers) {
                originalRequest.headers.Authorization = `Bearer ${token}`;
              }
              resolve(apiClient(originalRequest));
            },
            reject,
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const { refreshToken } = useAuthStore.getState();
        if (!refreshToken) {
          throw new Error('无 refresh token');
        }

        const refreshResponse = await axios.post('/api/v1/auth/refresh', {
          refresh_token: refreshToken,
        });

        const refreshData = refreshResponse.data as { code: number; data: { access_token: string; refresh_token: string } };
        if (refreshData.code !== 0) {
          throw new Error('刷新失败');
        }

        const { access_token, refresh_token } = refreshData.data;
        useAuthStore.getState().setTokens(access_token, refresh_token);

        processQueue(null, access_token);

        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
        }
        return apiClient(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError as Error, null);
        useAuthStore.getState().logout();
        message.error('登录已过期，请重新登录');
        window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`;
        return Promise.reject(new ApiError(401, '登录已过期'));
      } finally {
        isRefreshing = false;
      }
    }

    // 403 — 无权限
    if (status === 403) {
      message.error('无权限执行此操作');
      return Promise.reject(new ApiError(403, body?.message || '无权限执行此操作'));
    }

    // 429 — 限流
    if (status === 429) {
      message.warning('请求频繁，请稍后重试');
      return Promise.reject(new ApiError(429, '请求频繁，请稍后重试'));
    }

    const errMsg = body?.message || `请求失败 (${status})`;
    return Promise.reject(new ApiError(body?.code ?? status, errMsg));
  }
);

export default apiClient;