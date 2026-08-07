// Типы домена для AQI-платформы

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: 'admin' | 'analyst' | 'operator' | 'public';
  is_active: boolean;
  created_at: string;
}

export interface Sensor {
  id: string;
  name: string;
  location_name: string;
  latitude: number;
  longitude: number;
  is_active: boolean;
  last_seen_at: string | null;
  created_at: string;
}

export interface Measurement {
  time: string;
  sensor_id: string;
  no2: number | null;
  o3: number | null;
  co: number | null;
  h2s: number | null;
  so2: number | null;
  pm25: number | null;
  temperature: number | null;
  humidity: number | null;
  pressure: number | null;
  wind_speed: number | null;
  wind_dir: number | null;
  aqi: number | null;
  aqi_category: string | null;
}

export interface ForecastPoint {
  id: string;
  name: string;
  district: string;
  latitude: number;
  longitude: number;
}

export interface Forecast {
  point_id: string;
  point_name: string;
  district: string;
  forecasted_at: string;
  horizon_hours: number;
  aqi: number;
  aqi_category: string;
  no2: number | null;
  o3: number | null;
  pm25: number | null;
}

export interface AuthResponse {
  access_token: string;
  token_type: string;
  expires_at: string;
  user: User;
}

export interface ApiError {
  error: string;
  code?: string;
}

// AQI цвета по категориям
export const AQI_COLORS: Record<string, string> = {
  good: '#00e400',
  moderate: '#ffff00',
  sensitive: '#ff7e00',
  unhealthy: '#ff0000',
  very_unhealthy: '#8f3f97',
  hazardous: '#7e0023',
  unknown: '#999999',
};

export const AQI_LABELS: Record<string, string> = {
  good: 'Хорошее',
  moderate: 'Умеренное',
  sensitive: 'Вредно чувствительным',
  unhealthy: 'Вредное',
  very_unhealthy: 'Очень вредное',
  hazardous: 'Опасное',
  unknown: 'Нет данных',
};
