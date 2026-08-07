// HTTP-клиент на базе axios с автоматическим добавлением JWT-токена.
import axios, { type AxiosInstance, type AxiosError } from 'axios';
import type { ApiError } from '../types';

const BASE_URL = import.meta.env.VITE_API_URL ?? '/api/v1';

function createClient(): AxiosInstance {
  const instance = axios.create({
    baseURL: BASE_URL,
    timeout: 15_000,
    headers: { 'Content-Type': 'application/json' },
  });

  // Добавляем JWT из localStorage в каждый запрос.
  instance.interceptors.request.use((config) => {
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });

  // Единая обработка ошибок: 401 → редирект на /login.
  instance.interceptors.response.use(
    (r) => r,
    (err: AxiosError<ApiError>) => {
      if (err.response?.status === 401) {
        localStorage.removeItem('access_token');
        window.location.href = '/login';
      }
      return Promise.reject(err);
    }
  );

  return instance;
}

export const apiClient = createClient();
