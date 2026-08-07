// Хук загружает данные текущего пользователя при монтировании,
// если в localStorage есть токен, но user ещё не установлен в стор.
import { useEffect } from 'react';
import { getMe } from '../api/users';
import { useAuthStore } from '../store/authStore';

export function useMe() {
  const { token, user, setAuth } = useAuthStore();

  useEffect(() => {
    if (!token || user) return;

    getMe()
      .then((me) => {
        // Переиспользуем токен из стора — он уже там есть.
        setAuth(me, token);
      })
      .catch(() => {
        // Токен невалидный — очищаем.
        useAuthStore.getState().clearAuth();
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}
