// Страница входа в систему с валидацией формы.
import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { login } from '../api/auth';
import { useAuthStore } from '../store/authStore';
import styles from './LoginPage.module.css';

export default function LoginPage() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');

    if (!email.trim() || !password) {
      setError('Заполните все поля');
      return;
    }

    setLoading(true);
    try {
      const resp = await login(email.trim(), password);
      setAuth(resp.user, resp.access_token);
      navigate('/dashboard', { replace: true });
    } catch (err: unknown) {
      const msg =
        err instanceof Error
          ? err.message
          : 'Неверный email или пароль';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.root}>
      <div className={styles.card}>
        <div className={styles.header}>
          <span className={styles.icon}>🌫️</span>
          <h1 className={styles.title}>AQI Кемерово</h1>
          <p className={styles.subtitle}>Мониторинг качества воздуха</p>
        </div>

        <form onSubmit={handleSubmit} className={styles.form}>
          <div className={styles.field}>
            <label htmlFor="email" className={styles.label}>
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={styles.input}
              placeholder="admin@example.com"
              disabled={loading}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="password" className={styles.label}>
              Пароль
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={styles.input}
              placeholder="••••••••"
              disabled={loading}
            />
          </div>

          {error && <div className={styles.error}>{error}</div>}

          <button
            type="submit"
            className={styles.btn}
            disabled={loading}
          >
            {loading ? 'Вход…' : 'Войти'}
          </button>
        </form>

        <p className={styles.footer}>
          Система мониторинга атмосферного воздуха г. Кемерово
        </p>
      </div>
    </div>
  );
}
