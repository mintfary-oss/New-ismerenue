// Основной Layout: сайдбар + контент + мобильное меню-бургер
import { useState, useEffect } from 'react';
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom';
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
  const location = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);

  // Закрываем меню при смене маршрута (для мобильных)
  useEffect(() => {
    setMenuOpen(false);
  }, [location.pathname]);

  // Закрываем меню при клике вне сайдбара
  useEffect(() => {
    if (!menuOpen) return;
    const handler = (e: MouseEvent) => {
      const sidebar = document.getElementById('sidebar');
      const burger = document.getElementById('burger-btn');
      if (sidebar && !sidebar.contains(e.target as Node) &&
          burger && !burger.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [menuOpen]);

  async function handleLogout() {
    try { await logout(); } catch { /* ignore */ }
    clearAuth();
    navigate('/login');
  }

  return (
    <div className={styles.root}>
      {/* ── Мобильная шапка (topbar) ── */}
      <header className={styles.topbar}>
        <button
          id="burger-btn"
          className={styles.burgerBtn}
          onClick={() => setMenuOpen((v) => !v)}
          aria-label="Открыть меню"
          aria-expanded={menuOpen}
        >
          {menuOpen ? '✕' : '☰'}
        </button>
        <span className={styles.topbarTitle}>🌫️ AQI Кемерово</span>
      </header>

      {/* ── Затемнение фона на мобильном при открытом меню ── */}
      {menuOpen && (
        <div
          className={styles.overlay}
          onClick={() => setMenuOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* ── Сайдбар ── */}
      <aside
        id="sidebar"
        className={`${styles.sidebar} ${menuOpen ? styles.sidebarOpen : ''}`}
      >
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
