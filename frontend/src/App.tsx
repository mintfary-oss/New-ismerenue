// Корневой компонент с роутингом React Router v6.
import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import { useMe } from './hooks/useMe';
import LoginPage from './pages/LoginPage';
import NotFoundPage from './pages/NotFoundPage';
import Layout from './components/Layout';

// Ленивая загрузка тяжёлых страниц (MapLibre GL ~1 МБ загружается только при переходе на карту)
const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const MapPage = lazy(() => import('./pages/MapPage'));
const AdminPage = lazy(() => import('./pages/AdminPage'));

/** Защищённый маршрут — редирект на /login если нет токена. */
function PrivateRoute({ children }: { children: React.ReactNode }) {
  const isAuth = useAuthStore((s) => s.isAuthenticated());
  return isAuth ? <>{children}</> : <Navigate to="/login" replace />;
}

/** Маршрут только для Admin. */
function AdminRoute({ children }: { children: React.ReactNode }) {
  const hasRole = useAuthStore((s) => s.hasRole);
  return hasRole('admin') ? <>{children}</> : <Navigate to="/" replace />;
}

export default function App() {
  useMe(); // восстанавливаем user из токена при перезагрузке

  return (
    <BrowserRouter>
      <Routes>
        {/* Публичный маршрут */}
        <Route path="/login" element={<LoginPage />} />

        {/* Защищённые маршруты */}
        <Route
          path="/"
          element={
            <PrivateRoute>
              <Layout />
            </PrivateRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route
            path="dashboard"
            element={
              <Suspense fallback={<div style={{ padding: 24, color: 'var(--color-text-muted)' }}>Загрузка…</div>}>
                <DashboardPage />
              </Suspense>
            }
          />
          <Route
            path="map"
            element={
              <Suspense fallback={<div style={{ padding: 24, color: 'var(--color-text-muted)' }}>Загрузка карты…</div>}>
                <MapPage />
              </Suspense>
            }
          />
          <Route
            path="admin"
            element={
              <AdminRoute>
                <Suspense fallback={<div style={{ padding: 24, color: 'var(--color-text-muted)' }}>Загрузка…</div>}>
                  <AdminPage />
                </Suspense>
              </AdminRoute>
            }
          />
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  );
}
