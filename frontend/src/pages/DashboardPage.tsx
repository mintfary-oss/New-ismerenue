// Главный дашборд: текущее AQI по датчикам, история и прогноз.
import { useEffect, useState } from 'react';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import type { Sensor, Measurement, Forecast } from '../types';
import { AQI_COLORS, AQI_LABELS } from '../types';
import { listSensors, getMeasurements } from '../api/sensors';
import { getCurrentForecast, getPointForecast, getForecastPoints } from '../api/forecast';
import type { CurrentForecastResponse } from '../api/forecast';
import styles from './DashboardPage.module.css';

// ─── Компонент карточки AQI ──────────────────────────────────────────────────

interface AqiCardProps {
  label: string;
  aqi: number | null;
  category: string | null;
  sub?: string;
}

function AqiCard({ label, aqi, category, sub }: AqiCardProps) {
  const cat = category ?? 'unknown';
  const color = AQI_COLORS[cat] ?? AQI_COLORS.unknown;
  return (
    <div className={styles.card} style={{ borderTopColor: color }}>
      <div className={styles.cardLabel}>{label}</div>
      <div className={styles.cardAqi} style={{ color }}>
        {aqi !== null ? Math.round(aqi) : '—'}
      </div>
      <div className={styles.cardCat}>{AQI_LABELS[cat] ?? '—'}</div>
      {sub && <div className={styles.cardSub}>{sub}</div>}
    </div>
  );
}

// ─── Форматирование времени для графиков ─────────────────────────────────────

function fmtTime(iso: string): string {
  const d = new Date(iso);
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
}

function fmtDate(iso: string): string {
  const d = new Date(iso);
  return `${d.getDate()}.${(d.getMonth() + 1).toString().padStart(2, '0')} ${fmtTime(iso)}`;
}

// ─── DashboardPage ────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [selectedSensor, setSelectedSensor] = useState<string>('');
  const [history, setHistory] = useState<Measurement[]>([]);
  const [forecast, setForecast] = useState<CurrentForecastResponse | null>(null);
  const [pointForecasts, setPointForecasts] = useState<Forecast[]>([]);
  const [loading, setLoading] = useState(true);
  const [histLoading, setHistLoading] = useState(false);
  const [error, setError] = useState('');

  // Первичная загрузка: датчики + общий прогноз
  useEffect(() => {
    async function load() {
      try {
        const [sensorList, fc, points] = await Promise.all([
          listSensors(),
          getCurrentForecast(),
          getForecastPoints(),
        ]);
        setSensors(sensorList);
        setForecast(fc);

        // Берём первую точку прогноза для графика
        if (points.length > 0) {
          const pf = await getPointForecast(points[0].id);
          setPointForecasts(pf);
        }

        if (sensorList.length > 0) {
          setSelectedSensor(sensorList[0].id);
        }
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Ошибка загрузки');
      } finally {
        setLoading(false);
      }
    }
    void load();
  }, []);

  // Загрузка истории при смене датчика
  useEffect(() => {
    if (!selectedSensor) return;

    setHistLoading(true);
    const to = new Date().toISOString();
    const from = new Date(Date.now() - 24 * 3600 * 1000).toISOString();

    getMeasurements({ sensor_id: selectedSensor, from, to, limit: 200 })
      .then(setHistory)
      .catch(() => setHistory([]))
      .finally(() => setHistLoading(false));
  }, [selectedSensor]);

  // Данные для графика истории
  const histChartData = history.map((m) => ({
    time: fmtTime(m.time),
    AQI: m.aqi !== null ? Math.round(m.aqi) : null,
    NO2: m.no2,
    O3: m.o3,
    PM25: m.pm25,
    SO2: m.so2,
  }));

  // Данные для графика прогноза
  const forecastChartData = pointForecasts.map((f) => ({
    time: `+${f.horizon_hours}ч`,
    AQI: Math.round(f.aqi),
    label: fmtDate(f.forecasted_at),
  }));

  if (loading) {
    return <div className={styles.center}>Загрузка данных…</div>;
  }

  if (error) {
    return <div className={styles.center} style={{ color: '#fc8181' }}>{error}</div>;
  }

  return (
    <div className={styles.root}>
      <h2 className={styles.pageTitle}>Дашборд мониторинга</h2>

      {/* ── Карточки AQI ── */}
      <section>
        <h3 className={styles.sectionTitle}>Текущее состояние воздуха</h3>
        <div className={styles.cards}>
          {forecast?.city_average && (
            <AqiCard
              label="Город в целом"
              aqi={forecast.city_average.aqi}
              category={forecast.city_average.aqi_category}
              sub="прогноз"
            />
          )}
          {forecast?.points?.slice(0, 4).map((p) => (
            <AqiCard
              key={p.point_id}
              label={p.point_name}
              aqi={p.aqi}
              category={p.aqi_category}
              sub={p.district}
            />
          ))}
        </div>
      </section>

      {/* ── История датчика ── */}
      <section>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>История измерений (24 ч)</h3>
          <select
            className={styles.select}
            value={selectedSensor}
            onChange={(e) => setSelectedSensor(e.target.value)}
          >
            {sensors.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </div>

        {histLoading ? (
          <div className={styles.chartPlaceholder}>Загрузка…</div>
        ) : history.length === 0 ? (
          <div className={styles.chartPlaceholder}>Нет данных за последние 24 часа</div>
        ) : (
          <div className={styles.chartWrap}>
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={histChartData} margin={{ top: 5, right: 16, bottom: 0, left: 0 }}>
                <defs>
                  <linearGradient id="aqiGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#5b8dee" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#5b8dee" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#2d3148" />
                <XAxis dataKey="time" stroke="#8892a4" tick={{ fontSize: 11 }} />
                <YAxis stroke="#8892a4" tick={{ fontSize: 11 }} />
                <Tooltip
                  contentStyle={{ background: '#1a1d27', border: '1px solid #2d3148', borderRadius: 6 }}
                  labelStyle={{ color: '#e2e8f0' }}
                />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                <Area
                  type="monotone"
                  dataKey="AQI"
                  stroke="#5b8dee"
                  fill="url(#aqiGrad)"
                  strokeWidth={2}
                  dot={false}
                  connectNulls
                />
                <Line type="monotone" dataKey="NO2" stroke="#f6ad55" strokeWidth={1.5} dot={false} connectNulls />
                <Line type="monotone" dataKey="PM25" stroke="#fc8181" strokeWidth={1.5} dot={false} connectNulls />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </section>

      {/* ── Прогноз ── */}
      {forecastChartData.length > 0 && (
        <section>
          <h3 className={styles.sectionTitle}>Прогноз AQI (6 часов)</h3>
          <div className={styles.chartWrap}>
            <ResponsiveContainer width="100%" height={180}>
              <LineChart data={forecastChartData} margin={{ top: 5, right: 16, bottom: 0, left: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2d3148" />
                <XAxis dataKey="time" stroke="#8892a4" tick={{ fontSize: 11 }} />
                <YAxis stroke="#8892a4" tick={{ fontSize: 11 }} />
                <Tooltip
                  contentStyle={{ background: '#1a1d27', border: '1px solid #2d3148', borderRadius: 6 }}
                  labelStyle={{ color: '#e2e8f0' }}
                />
                <Line
                  type="monotone"
                  dataKey="AQI"
                  stroke="#68d391"
                  strokeWidth={2}
                  strokeDasharray="5 3"
                  dot={{ fill: '#68d391', r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </section>
      )}

      {/* ── Сводная таблица датчиков ── */}
      <section>
        <h3 className={styles.sectionTitle}>Датчики</h3>
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Название</th>
                <th>Район / адрес</th>
                <th>Статус</th>
                <th>Последняя передача</th>
              </tr>
            </thead>
            <tbody>
              {sensors.map((s) => (
                <tr key={s.id}>
                  <td>{s.name}</td>
                  <td>{s.location_name}</td>
                  <td>
                    <span className={s.is_active ? styles.badgeGreen : styles.badgeRed}>
                      {s.is_active ? 'Активен' : 'Отключён'}
                    </span>
                  </td>
                  <td className={styles.muted}>
                    {s.last_seen_at
                      ? new Date(s.last_seen_at).toLocaleString('ru-RU')
                      : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
