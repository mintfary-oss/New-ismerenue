// Административная панель: управление пользователями, датчиками и отчётами.
// Доступна только пользователям с ролью admin.
import { useEffect, useState } from 'react';
import type { User, Sensor } from '../types';
import { listUsers, createUser, deleteUser } from '../api/users';
import {
  listSensors,
  createSensor,
  updateSensor,
  deleteSensor,
} from '../api/sensors';
import { apiClient } from '../api/client';
import styles from './AdminPage.module.css';

// ─── Типы ────────────────────────────────────────────────────────────────────

interface Report {
  id: string;
  name: string;
  type: string;
  status: string;
  created_at: string;
  download_url: string | null;
}

// ─── Утилиты ─────────────────────────────────────────────────────────────────

function fmtDate(iso: string) {
  return new Date(iso).toLocaleString('ru-RU');
}

// ─── Подкомпонент: вкладка пользователей ──────────────────────────────────────

function UsersTab() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ email: '', password: '', full_name: '', role: 'analyst' });
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState('');

  async function load() {
    setLoading(true);
    try {
      setUsers(await listUsers());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setErr('');
    try {
      await createUser(form);
      setShowForm(false);
      setForm({ email: '', password: '', full_name: '', role: 'analyst' });
      await load();
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : 'Ошибка создания');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string, email: string) {
    if (!confirm(`Удалить пользователя ${email}?`)) return;
    try {
      await deleteUser(id);
      await load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'Ошибка удаления');
    }
  }

  const ROLES = ['admin', 'analyst', 'operator', 'public'];

  return (
    <div className={styles.tabContent}>
      <div className={styles.tabHeader}>
        <span className={styles.count}>{users.length} пользователей</span>
        <button className={styles.btnPrimary} onClick={() => setShowForm((v) => !v)}>
          {showForm ? 'Отмена' : '+ Добавить'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className={styles.form}>
          <h4 className={styles.formTitle}>Новый пользователь</h4>
          <div className={styles.formGrid}>
            <input
              className={styles.input}
              placeholder="Email"
              type="email"
              required
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
            <input
              className={styles.input}
              placeholder="Пароль"
              type="password"
              required
              minLength={8}
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
            <input
              className={styles.input}
              placeholder="Полное имя"
              required
              value={form.full_name}
              onChange={(e) => setForm({ ...form, full_name: e.target.value })}
            />
            <select
              className={styles.select}
              value={form.role}
              onChange={(e) => setForm({ ...form, role: e.target.value })}
            >
              {ROLES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          {err && <div className={styles.err}>{err}</div>}
          <button type="submit" className={styles.btnPrimary} disabled={saving}>
            {saving ? 'Сохранение…' : 'Создать'}
          </button>
        </form>
      )}

      {loading ? (
        <div className={styles.placeholder}>Загрузка…</div>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Email</th>
                <th>Имя</th>
                <th>Роль</th>
                <th>Статус</th>
                <th>Создан</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>{u.email}</td>
                  <td>{u.full_name}</td>
                  <td><span className={styles.badge}>{u.role}</span></td>
                  <td>
                    <span className={u.is_active ? styles.badgeGreen : styles.badgeRed}>
                      {u.is_active ? 'Активен' : 'Заблокирован'}
                    </span>
                  </td>
                  <td className={styles.muted}>{fmtDate(u.created_at)}</td>
                  <td>
                    <button
                      className={styles.btnDanger}
                      onClick={() => void handleDelete(u.id, u.email)}
                    >
                      Удалить
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ─── Подкомпонент: вкладка датчиков ───────────────────────────────────────────

function SensorsTab() {
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [form, setForm] = useState({
    name: '',
    location_name: '',
    latitude: '',
    longitude: '',
    is_active: true,
  });
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState('');

  async function load() {
    setLoading(true);
    try {
      setSensors(await listSensors());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, []);

  function openCreate() {
    setEditId(null);
    setForm({ name: '', location_name: '', latitude: '', longitude: '', is_active: true });
    setShowForm(true);
    setErr('');
  }

  function openEdit(s: Sensor) {
    setEditId(s.id);
    setForm({
      name: s.name,
      location_name: s.location_name,
      latitude: String(s.latitude),
      longitude: String(s.longitude),
      is_active: s.is_active,
    });
    setShowForm(true);
    setErr('');
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setErr('');
    const payload = {
      name: form.name,
      location_name: form.location_name,
      latitude: parseFloat(form.latitude),
      longitude: parseFloat(form.longitude),
      is_active: form.is_active,
    };
    try {
      if (editId) {
        await updateSensor(editId, payload);
      } else {
        await createSensor(payload);
      }
      setShowForm(false);
      await load();
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : 'Ошибка сохранения');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string, name: string) {
    if (!confirm(`Удалить датчик "${name}"?`)) return;
    try {
      await deleteSensor(id);
      await load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'Ошибка удаления');
    }
  }

  return (
    <div className={styles.tabContent}>
      <div className={styles.tabHeader}>
        <span className={styles.count}>{sensors.length} датчиков</span>
        <button className={styles.btnPrimary} onClick={openCreate}>
          + Добавить датчик
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSave} className={styles.form}>
          <h4 className={styles.formTitle}>
            {editId ? 'Редактировать датчик' : 'Новый датчик'}
          </h4>
          <div className={styles.formGrid}>
            <input
              className={styles.input}
              placeholder="Название"
              required
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
            <input
              className={styles.input}
              placeholder="Адрес / район"
              required
              value={form.location_name}
              onChange={(e) => setForm({ ...form, location_name: e.target.value })}
            />
            <input
              className={styles.input}
              placeholder="Широта (55.xxxx)"
              type="number"
              step="0.0001"
              required
              value={form.latitude}
              onChange={(e) => setForm({ ...form, latitude: e.target.value })}
            />
            <input
              className={styles.input}
              placeholder="Долгота (86.xxxx)"
              type="number"
              step="0.0001"
              required
              value={form.longitude}
              onChange={(e) => setForm({ ...form, longitude: e.target.value })}
            />
          </div>
          <label className={styles.checkLabel}>
            <input
              type="checkbox"
              checked={form.is_active}
              onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
            />
            Активен
          </label>
          {err && <div className={styles.err}>{err}</div>}
          <div className={styles.formActions}>
            <button type="submit" className={styles.btnPrimary} disabled={saving}>
              {saving ? 'Сохранение…' : editId ? 'Сохранить' : 'Создать'}
            </button>
            <button type="button" className={styles.btnGhost} onClick={() => setShowForm(false)}>
              Отмена
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <div className={styles.placeholder}>Загрузка…</div>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Название</th>
                <th>Адрес</th>
                <th>Координаты</th>
                <th>Статус</th>
                <th>Последний сигнал</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {sensors.map((s) => (
                <tr key={s.id}>
                  <td>{s.name}</td>
                  <td>{s.location_name}</td>
                  <td className={styles.muted}>
                    {s.latitude.toFixed(4)}, {s.longitude.toFixed(4)}
                  </td>
                  <td>
                    <span className={s.is_active ? styles.badgeGreen : styles.badgeRed}>
                      {s.is_active ? 'Активен' : 'Откл.'}
                    </span>
                  </td>
                  <td className={styles.muted}>
                    {s.last_seen_at ? fmtDate(s.last_seen_at) : '—'}
                  </td>
                  <td className={styles.actions}>
                    <button className={styles.btnGhost} onClick={() => openEdit(s)}>
                      Изм.
                    </button>
                    <button
                      className={styles.btnDanger}
                      onClick={() => void handleDelete(s.id, s.name)}
                    >
                      Удалить
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ─── Подкомпонент: вкладка отчётов ────────────────────────────────────────────

function ReportsTab() {
  const [reports, setReports] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [form, setForm] = useState({ name: '', type: 'daily', from: '', to: '' });

  async function load() {
    setLoading(true);
    try {
      const { data } = await apiClient.get<{ reports: Report[] }>('/reports');
      setReports(data.reports ?? []);
    } catch {
      setReports([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, []);

  async function handleGenerate(e: React.FormEvent) {
    e.preventDefault();
    setGenerating(true);
    try {
      await apiClient.post('/reports', form);
      await load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'Ошибка генерации отчёта');
    } finally {
      setGenerating(false);
    }
  }

  const REPORT_TYPES = [
    { value: 'daily', label: 'Ежедневный' },
    { value: 'weekly', label: 'Еженедельный' },
    { value: 'monthly', label: 'Ежемесячный' },
    { value: 'custom', label: 'Произвольный период' },
  ];

  return (
    <div className={styles.tabContent}>
      <div className={styles.tabHeader}>
        <span className={styles.count}>Отчёты</span>
      </div>

      <form onSubmit={handleGenerate} className={styles.form}>
        <h4 className={styles.formTitle}>Сгенерировать отчёт</h4>
        <div className={styles.formGrid}>
          <input
            className={styles.input}
            placeholder="Название отчёта"
            required
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <select
            className={styles.select}
            value={form.type}
            onChange={(e) => setForm({ ...form, type: e.target.value })}
          >
            {REPORT_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
          <input
            className={styles.input}
            type="date"
            required
            value={form.from}
            onChange={(e) => setForm({ ...form, from: e.target.value })}
            title="Дата начала"
          />
          <input
            className={styles.input}
            type="date"
            required
            value={form.to}
            onChange={(e) => setForm({ ...form, to: e.target.value })}
            title="Дата окончания"
          />
        </div>
        <button type="submit" className={styles.btnPrimary} disabled={generating}>
          {generating ? 'Генерация…' : 'Сгенерировать CSV'}
        </button>
      </form>

      {loading ? (
        <div className={styles.placeholder}>Загрузка…</div>
      ) : reports.length === 0 ? (
        <div className={styles.placeholder}>Нет созданных отчётов</div>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Название</th>
                <th>Тип</th>
                <th>Статус</th>
                <th>Создан</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {reports.map((r) => (
                <tr key={r.id}>
                  <td>{r.name}</td>
                  <td className={styles.muted}>{r.type}</td>
                  <td>
                    <span className={r.status === 'ready' ? styles.badgeGreen : styles.badge}>
                      {r.status}
                    </span>
                  </td>
                  <td className={styles.muted}>{fmtDate(r.created_at)}</td>
                  <td>
                    {r.download_url && (
                      <a
                        href={r.download_url}
                        className={styles.btnPrimary}
                        download
                        style={{ textDecoration: 'none', display: 'inline-block' }}
                      >
                        Скачать
                      </a>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ─── Главный компонент AdminPage ───────────────────────────────────────────────

type TabId = 'users' | 'sensors' | 'reports';

const TABS: { id: TabId; label: string; icon: string }[] = [
  { id: 'users', label: 'Пользователи', icon: '👥' },
  { id: 'sensors', label: 'Датчики', icon: '📡' },
  { id: 'reports', label: 'Отчёты', icon: '📄' },
];

export default function AdminPage() {
  const [tab, setTab] = useState<TabId>('users');

  return (
    <div className={styles.root}>
      <h2 className={styles.pageTitle}>Администрирование</h2>

      <div className={styles.tabs}>
        {TABS.map((t) => (
          <button
            key={t.id}
            className={`${styles.tabBtn} ${tab === t.id ? styles.tabActive : ''}`}
            onClick={() => setTab(t.id)}
          >
            <span>{t.icon}</span>
            <span>{t.label}</span>
          </button>
        ))}
      </div>

      {tab === 'users' && <UsersTab />}
      {tab === 'sensors' && <SensorsTab />}
      {tab === 'reports' && <ReportsTab />}
    </div>
  );
}
