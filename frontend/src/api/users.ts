import { apiClient } from './client';
import type { User } from '../types';

export async function listUsers(): Promise<User[]> {
  const { data } = await apiClient.get<{ users: User[] }>('/users');
  return data.users ?? [];
}

export async function createUser(payload: {
  email: string;
  password: string;
  full_name: string;
  role: string;
}): Promise<User> {
  const { data } = await apiClient.post<User>('/users', payload);
  return data;
}

export async function updateUser(id: string, payload: Partial<User>): Promise<User> {
  const { data } = await apiClient.put<User>(`/users/${id}`, payload);
  return data;
}

export async function deleteUser(id: string): Promise<void> {
  await apiClient.delete(`/users/${id}`);
}

export async function getMe(): Promise<User> {
  const { data } = await apiClient.get<User>('/users/me');
  return data;
}
