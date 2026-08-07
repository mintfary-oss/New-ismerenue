// Основной Layout: сайдбар + контент
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { logout } from '../api/auth';
import styles from './Layout.module.css';

const NAV_ITEMS = [
  { path: '/dashboard', label: 'Дашборд', icon: '📊' },
  { path: '/map', label: 'Карта', icon: '🗺️' },
  { path: '/howto', label: 'Как пользоваться', icon: '📖' },
];

const ADMIN_ITEMS = [
  { path: '/admin', label: 'Администрирование', icon: '⚙️' },
];

export default function Layout() {
  const { user, clearAuth, hasRole } = useAuthStore();
  const navigate = useNavigate();

  async function handleLogout() {
    try { await logout(); } catch { /* ignore */ }
    clearAuth();
    navigate('/login');
  }

  return (
    <div className={styles.root}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>
          <span className={styles.logoIcon}>🌫️</span>
          <span className={styles.logoText}>AQI Кемерово</span>
        </div>

        <nav className={styles.nav}>
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `${styles.navItem} ${isActive ? styles.navActive : ''}`
              }
            >
              <span className={styles.navIcon}>{item.icon}</span>
              <span>{item.label}</span>
            </NavLink>
          ))}

          {hasRole('admin') &&
            ADMIN_ITEMS.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  `${styles.navItem} ${isActive ? styles.navActive : ''}`
                }
              >
                <span className={styles.navIcon}>{item.icon}</span>
                <span>{item.label}</span>
              </NavLink>
            ))}
        </nav>

        <div className={styles.userBlock}>
          <div className={styles.userName}>{user?.full_name ?? user?.email}</div>
          <div className={styles.userRole}>{user?.role}</div>
          <button className={styles.logoutBtn} onClick={handleLogout}>
            Выйти
          </button>
        </div>
      </aside>

      <main className={styles.content}>
        <Outlet />
      </main>
    </div>
  );
}
