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

// ─── Подкомпонент: вкладка статистики ─────────────────────────────────────────

interface AvailabilityStat {
  sensor_id: string;
  sensor_name: string;
  total_hours: number;
  online_hours: number;
  availability_percent: number;
}

interface DataCoverageStat {
  sensor_id: string;
  parameter: string;
  coverage_percent: number;
  total_records: number;
}

function StatsTab() {
  const [availability, setAvailability] = useState<AvailabilityStat[]>([]);
  const [coverage, setCoverage] = useState<DataCoverageStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');

  useEffect(() => {
    void (async () => {
      setLoading(true);
      setErr('');
      try {
        const [avRes, covRes] = await Promise.all([
          apiClient.get<AvailabilityStat[] | { items: AvailabilityStat[] }>('/stats/availability'),
          apiClient.get<DataCoverageStat[] | { items: DataCoverageStat[] }>('/stats/data-coverage'),
        ]);
        const avData = Array.isArray(avRes.data) ? avRes.data : (avRes.data as { items: AvailabilityStat[] }).items ?? [];
        const covData = Array.isArray(covRes.data) ? covRes.data : (covRes.data as { items: DataCoverageStat[] }).items ?? [];
        setAvailability(avData);
        setCoverage(covData);
      } catch (e: unknown) {
        setErr(e instanceof Error ? e.message : 'Ошибка загрузки статистики');
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  function pctColor(v: number): string {
    if (v >= 90) return '#68d391';
    if (v >= 70) return '#f6e05e';
    return '#fc8181';
  }

  function PctBar({ value }: { value: number }) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <div style={{
          width: 80, height: 6, background: 'var(--color-border)', borderRadius: 3, overflow: 'hidden',
        }}>
          <div style={{
            width: `${Math.min(100, value)}%`, height: '100%',
            background: pctColor(value), borderRadius: 3, transition: 'width 0.4s',
          }} />
        </div>
        <span style={{ fontSize: 12, color: pctColor(value), fontWeight: 600 }}>
          {value.toFixed(1)}%
        </span>
      </div>
    );
  }

  if (loading) return <div className={styles.placeholder}>Загрузка статистики…</div>;
  if (err) return <div className={styles.placeholder} style={{ color: '#fc8181' }}>{err}</div>;

  return (
    <div className={styles.tabContent}>
      <div className={styles.tabHeader}>
        <span className={styles.count}>Доступность датчиков</span>
      </div>

      {availability.length === 0 ? (
        <div className={styles.placeholder}>Нет данных доступности (нужны исторические данные)</div>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Датчик</th>
                <th>Часов онлайн</th>
                <th>Всего часов</th>
                <th>Доступность</th>
              </tr>
            </thead>
            <tbody>
              {availability.map((a) => (
                <tr key={a.sensor_id}>
                  <td>{a.sensor_name}</td>
                  <td className={styles.muted}>{a.online_hours}</td>
                  <td className={styles.muted}>{a.total_hours}</td>
                  <td><PctBar value={a.availability_percent} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className={styles.tabHeader} style={{ marginTop: 8 }}>
        <span className={styles.count}>Покрытие данных по параметрам</span>
      </div>

      {coverage.length === 0 ? (
        <div className={styles.placeholder}>Нет данных покрытия</div>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Датчик</th>
                <th>Параметр</th>
                <th>Записей</th>
                <th>Покрытие</th>
              </tr>
            </thead>
            <tbody>
              {coverage.map((c, i) => (
                <tr key={`${c.sensor_id}-${c.parameter}-${i}`}>
                  <td className={styles.muted} style={{ fontSize: 11 }}>{c.sensor_id.slice(0, 8)}…</td>
                  <td><code style={{ fontSize: 11 }}>{c.parameter}</code></td>
                  <td className={styles.muted}>{c.total_records}</td>
                  <td><PctBar value={c.coverage_percent} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ─── Подкомпонент: вкладка системной информации ────────────────────────────────

interface HealthInfo {
  status: string;
  dependencies?: Record<string, { status: string; latency?: string; error?: string }>;
  uptime?: string;
  go_version?: string;
  num_cpu?: number;
  goroutines?: number;
}

function SystemTab() {
  const [health, setHealth] = useState<HealthInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshedAt, setRefreshedAt] = useState(new Date());

  async function load() {
    setLoading(true);
    try {
      const res = await apiClient.get<HealthInfo>('/../../ready');
      setHealth(res.data);
      setRefreshedAt(new Date());
    } catch {
      setHealth({ status: 'error' });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, []);

  const STATUS_LABELS: Record<string, { label: string; cls: string }> = {
    ok: { label: 'OK', cls: styles.badgeGreen },
    degraded: { label: 'Деградация', cls: styles.badgeRed },
    error: { label: 'Ошибка', cls: styles.badgeRed },
    not_configured: { label: 'Не настроено', cls: styles.badge },
  };

  function StatusBadge({ status }: { status: string }) {
    const s = STATUS_LABELS[status] ?? { label: status, cls: styles.badge };
    return <span className={s.cls}>{s.label}</span>;
  }

  const INFO_ROWS = [
    { label: 'Версия платформы', value: 'AQI Platform v1.0.0' },
    { label: 'Язык / Runtime', value: health?.go_version ?? '—' },
    { label: 'CPU cores', value: health?.num_cpu != null ? String(health.num_cpu) : '—' },
    { label: 'Goroutines', value: health?.goroutines != null ? String(health.goroutines) : '—' },
    { label: 'Uptime', value: health?.uptime ?? '—' },
    { label: 'Статус проверен', value: refreshedAt.toLocaleString('ru-RU') },
  ];

  return (
    <div className={styles.tabContent}>
      <div className={styles.tabHeader}>
        <span className={styles.count}>Состояние системы</span>
        <button className={styles.btnGhost} onClick={() => void load()} disabled={loading}>
          {loading ? 'Проверка…' : '↺ Обновить'}
        </button>
      </div>

      {/* Общий статус */}
      <div style={{
        background: 'var(--color-surface)', border: '1px solid var(--color-border)',
        borderRadius: 'var(--radius)', padding: '16px 20px',
        display: 'flex', alignItems: 'center', gap: 12,
      }}>
        <span style={{ fontSize: 24 }}>{health?.status === 'ok' ? '✅' : '⚠️'}</span>
        <div>
          <div style={{ fontWeight: 600, marginBottom: 2 }}>
            {health?.status === 'ok' ? 'Всё работает нормально' : 'Обнаружены проблемы'}
          </div>
          {health && <StatusBadge status={health.status} />}
        </div>
      </div>

      {/* Зависимости */}
      {health?.dependencies && Object.keys(health.dependencies).length > 0 && (
        <>
          <div className={styles.tabHeader} style={{ marginTop: 4 }}>
            <span className={styles.count}>Зависимости</span>
          </div>
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Сервис</th>
                  <th>Статус</th>
                  <th>Задержка</th>
                  <th>Ошибка</th>
                </tr>
              </thead>
              <tbody>
                {Object.entries(health.dependencies).map(([name, dep]) => (
                  <tr key={name}>
                    <td style={{ fontWeight: 500 }}>{name}</td>
                    <td><StatusBadge status={dep.status} /></td>
                    <td className={styles.muted}>{dep.latency ?? '—'}</td>
                    <td className={styles.muted} style={{ color: dep.error ? '#fc8181' : undefined }}>
                      {dep.error ?? '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* Системная информация */}
      <div className={styles.tabHeader} style={{ marginTop: 4 }}>
        <span className={styles.count}>Системная информация</span>
      </div>
      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <tbody>
            {INFO_ROWS.map((row) => (
              <tr key={row.label}>
                <td style={{ fontWeight: 500, width: '40%' }}>{row.label}</td>
                <td className={styles.muted}>{row.value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Ссылки */}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <a
          href="/api/v1/docs"
          target="_blank"
          rel="noopener noreferrer"
          className={styles.btnGhost}
          style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 4 }}
        >
          📖 Swagger API Docs
        </a>
        <a
          href="/api/v1/openapi.yaml"
          target="_blank"
          rel="noopener noreferrer"
          className={styles.btnGhost}
          style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 4 }}
        >
          📄 OpenAPI YAML
        </a>
        <a
          href="/widget/"
          target="_blank"
          rel="noopener noreferrer"
          className={styles.btnGhost}
          style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 4 }}
        >
          🌐 Публичный виджет
        </a>
      </div>
    </div>
  );
}

// ─── Главный компонент AdminPage ───────────────────────────────────────────────

type TabId = 'users' | 'sensors' | 'reports' | 'stats' | 'system';

const TABS: { id: TabId; label: string; icon: string }[] = [
  { id: 'users', label: 'Пользователи', icon: '👥' },
  { id: 'sensors', label: 'Датчики', icon: '📡' },
  { id: 'reports', label: 'Отчёты', icon: '📄' },
  { id: 'stats', label: 'Статистика', icon: '📈' },
  { id: 'system', label: 'Система', icon: '⚙️' },
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
      {tab === 'stats' && <StatsTab />}
      {tab === 'system' && <SystemTab />}
    </div>
  );
}
