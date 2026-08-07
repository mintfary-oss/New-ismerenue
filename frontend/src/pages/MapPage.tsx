// Страница карты с датчиками качества воздуха.
// Использует MapLibre GL 6 (BSD) — открытый форк Mapbox GL.
// Тайлы: OpenStreetMap через стандартный растровый стиль.
import { useEffect, useRef, useState } from 'react';
import * as maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import type { Sensor, Measurement } from '../types';
import { AQI_COLORS, AQI_LABELS } from '../types';
import { listSensors, getLatestMeasurements } from '../api/sensors';
import { getCurrentForecast } from '../api/forecast';
import type { CurrentForecastResponse } from '../api/forecast';
import styles from './MapPage.module.css';

// Кемерово: центр карты
const KEMEROVO = { lng: 86.0872, lat: 55.3544 };

// Стиль карты через OpenStreetMap (бесплатно, без ключа)
const MAP_STYLE = {
  version: 8 as const,
  sources: {
    osm: {
      type: 'raster' as const,
      tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
      tileSize: 256,
      attribution: '© OpenStreetMap contributors',
      maxzoom: 19,
    },
  },
  layers: [
    {
      id: 'osm-tiles',
      type: 'raster' as const,
      source: 'osm',
    },
  ],
};

interface SensorWithAqi extends Sensor {
  aqi: number | null;
  aqi_category: string | null;
}

export default function MapPage() {
  const mapContainer = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const markersRef = useRef<maplibregl.Marker[]>([]);

  const [sensors, setSensors] = useState<SensorWithAqi[]>([]);
  const [forecast, setForecast] = useState<CurrentForecastResponse | null>(null);
  const [selected, setSelected] = useState<SensorWithAqi | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Инициализируем карту
  useEffect(() => {
    if (!mapContainer.current || mapRef.current) return;

    const map = new maplibregl.Map({
      container: mapContainer.current,
      style: MAP_STYLE,
      center: [KEMEROVO.lng, KEMEROVO.lat],
      zoom: 11,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');
    map.addControl(
      new maplibregl.ScaleControl({ maxWidth: 100, unit: 'metric' }),
      'bottom-left'
    );

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  // Загружаем датчики + последние измерения
  useEffect(() => {
    async function load() {
      setLoading(true);
      try {
        const [rawSensors, measurements, fc] = await Promise.all([
          listSensors(),
          getLatestMeasurements(),
          getCurrentForecast(),
        ]);

        // Склеиваем датчики с последними AQI-данными
        const measureMap = new Map<string, Measurement>();
        for (const m of measurements) {
          measureMap.set(m.sensor_id, m);
        }

        const enriched: SensorWithAqi[] = rawSensors.map((s) => {
          const m = measureMap.get(s.id);
          return {
            ...s,
            aqi: m?.aqi ?? null,
            aqi_category: m?.aqi_category ?? null,
          };
        });

        setSensors(enriched);
        setForecast(fc);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Ошибка загрузки данных');
      } finally {
        setLoading(false);
      }
    }
    void load();
  }, []);

  // Добавляем маркеры после загрузки датчиков
  useEffect(() => {
    const map = mapRef.current;
    if (!map || sensors.length === 0) return;

    // Ждём пока карта загрузится
    function addMarkers() {
      // Удаляем старые маркеры
      markersRef.current.forEach((m) => m.remove());
      markersRef.current = [];

      for (const sensor of sensors) {
        const color =
          AQI_COLORS[sensor.aqi_category ?? 'unknown'] ?? AQI_COLORS.unknown;
        const label = sensor.aqi !== null ? String(Math.round(sensor.aqi)) : '?';

        // Создаём кастомный элемент маркера
        const el = document.createElement('div');
        el.className = styles.marker;
        el.style.backgroundColor = color;
        el.title = `${sensor.name}\nAQI: ${label}`;

        const span = document.createElement('span');
        span.textContent = label;
        el.appendChild(span);

        el.addEventListener('click', () => {
          setSelected(sensor);
        });

        const marker = new maplibregl.Marker({ element: el, anchor: 'center' })
          .setLngLat([sensor.longitude, sensor.latitude])
          .addTo(map!);

        markersRef.current.push(marker);
      }
    }

    if (map.isStyleLoaded()) {
      addMarkers();
    } else {
      map.once('load', addMarkers);
    }
  }, [sensors]);

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <h2 className={styles.title}>Карта датчиков</h2>
        {forecast?.city_average && (
          <div
            className={styles.cityAqi}
            style={{ borderColor: AQI_COLORS[forecast.city_average.aqi_category] }}
          >
            <span className={styles.cityAqiLabel}>Город в целом:</span>
            <span
              className={styles.cityAqiValue}
              style={{ color: AQI_COLORS[forecast.city_average.aqi_category] }}
            >
              AQI {Math.round(forecast.city_average.aqi)}
            </span>
            <span className={styles.cityAqiCat}>
              {AQI_LABELS[forecast.city_average.aqi_category] ?? ''}
            </span>
          </div>
        )}
      </div>

      {loading && <div className={styles.loading}>Загрузка данных…</div>}
      {error && <div className={styles.error}>{error}</div>}

      <div className={styles.mapWrap}>
        <div ref={mapContainer} className={styles.map} />

        {/* Легенда */}
        <div className={styles.legend}>
          {Object.entries(AQI_LABELS).map(([key, label]) => (
            <div key={key} className={styles.legendItem}>
              <span
                className={styles.legendDot}
                style={{ backgroundColor: AQI_COLORS[key] }}
              />
              <span>{label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Карточка выбранного датчика */}
      {selected && (
        <div className={styles.popup}>
          <button
            className={styles.popupClose}
            onClick={() => setSelected(null)}
          >
            ✕
          </button>
          <h3 className={styles.popupName}>{selected.name}</h3>
          <p className={styles.popupLocation}>{selected.location_name}</p>
          <div className={styles.popupAqi}>
            <span
              style={{
                color: AQI_COLORS[selected.aqi_category ?? 'unknown'],
                fontSize: 32,
                fontWeight: 700,
              }}
            >
              {selected.aqi !== null ? Math.round(selected.aqi) : '—'}
            </span>
            <span className={styles.popupCat}>
              {AQI_LABELS[selected.aqi_category ?? 'unknown']}
            </span>
          </div>
          <p className={styles.popupCoords}>
            {selected.latitude.toFixed(4)}, {selected.longitude.toFixed(4)}
          </p>
          <p className={styles.popupStatus}>
            Статус: {selected.is_active ? '🟢 активен' : '🔴 отключён'}
          </p>
        </div>
      )}
    </div>
  );
}
