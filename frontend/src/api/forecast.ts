import { apiClient } from './client';
import type { Forecast, ForecastPoint } from '../types';

export async function getForecastPoints(): Promise<ForecastPoint[]> {
  const { data } = await apiClient.get<{ points: ForecastPoint[] }>('/forecast/points');
  return data.points ?? [];
}

export interface CurrentForecastResponse {
  city_average: Forecast | null;
  points: Forecast[];
}

export async function getCurrentForecast(): Promise<CurrentForecastResponse> {
  const { data } = await apiClient.get<CurrentForecastResponse>('/forecast/current');
  return data;
}

export async function getPointForecast(pointId: string): Promise<Forecast[]> {
  const { data } = await apiClient.get<{ forecasts: Forecast[] }>(
    `/forecast/point/${pointId}`
  );
  return data.forecasts ?? [];
}
