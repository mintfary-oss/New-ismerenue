import { apiClient } from './client';
import type { Sensor, Measurement } from '../types';

export async function listSensors(): Promise<Sensor[]> {
  const { data } = await apiClient.get<{ sensors: Sensor[] }>('/sensors');
  return data.sensors ?? [];
}

export async function getSensor(id: string): Promise<Sensor> {
  const { data } = await apiClient.get<Sensor>(`/sensors/${id}`);
  return data;
}

export async function createSensor(payload: Partial<Sensor>): Promise<Sensor> {
  const { data } = await apiClient.post<Sensor>('/sensors', payload);
  return data;
}

export async function updateSensor(id: string, payload: Partial<Sensor>): Promise<Sensor> {
  const { data } = await apiClient.put<Sensor>(`/sensors/${id}`, payload);
  return data;
}

export async function deleteSensor(id: string): Promise<void> {
  await apiClient.delete(`/sensors/${id}`);
}

export interface MeasurementsQuery {
  sensor_id?: string;
  from?: string;
  to?: string;
  limit?: number;
}

export async function getMeasurements(q: MeasurementsQuery): Promise<Measurement[]> {
  const { data } = await apiClient.get<{ measurements: Measurement[] }>('/measurements', {
    params: q,
  });
  return data.measurements ?? [];
}

export async function getLatestMeasurements(): Promise<Measurement[]> {
  const { data } = await apiClient.get<{ measurements: Measurement[] }>('/measurements/latest');
  return data.measurements ?? [];
}
