// Глобальное состояние аутентификации через Zustand.
import { create } from 'zustand';
import type { User } from '../types';

interface AuthState {
  user: User | null;
  token: string | null;
  setAuth: (user: User, token: string) => void;
  clearAuth: () => void;
  isAuthenticated: () => boolean;
  hasRole: (role: string | string[]) => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  token: localStorage.getItem('access_token'),

  setAuth: (user, token) => {
    localStorage.setItem('access_token', token);
    set({ user, token });
  },

  clearAuth: () => {
    localStorage.removeItem('access_token');
    set({ user: null, token: null });
  },

  isAuthenticated: () => {
    return get().token !== null;
  },

  hasRole: (role) => {
    const { user } = get();
    if (!user) return false;
    const roles = Array.isArray(role) ? role : [role];
    return roles.includes(user.role);
  },
}));
