import { DataSourceJsonData, DataQuery, SelectableValue } from '@grafana/data';

// ---------------------------------------------------------------------------
// Govee API types
// ---------------------------------------------------------------------------

export interface GoveeCapability {
  type: string;
  instance: string;
  parameters?: Record<string, unknown>;
}

export interface GoveeDevice {
  sku: string;
  device: string;
  deviceName: string;
  type: string;
  capabilities: GoveeCapability[];
}

export interface GoveeDeviceStateValue {
  instance: string;
  state: unknown;
  unit?: string;
}

export interface GoveeDeviceState {
  sku: string;
  device: string;
  capabilities: GoveeDeviceStateValue[];
}

// ---------------------------------------------------------------------------
// Well-known capability instances the plugin surfaces as metrics
// ---------------------------------------------------------------------------

export type MetricInstance =
  | 'temperature'
  | 'humidity'
  | 'battery'
  | 'powerState'
  | 'brightness'
  | 'colorTemperature'
  | 'online'
  | 'custom';

export const METRIC_OPTIONS: Array<SelectableValue<MetricInstance>> = [
  { label: 'Temperature', value: 'temperature' },
  { label: 'Humidity', value: 'humidity' },
  { label: 'Battery', value: 'battery' },
  { label: 'Power State', value: 'powerState' },
  { label: 'Brightness', value: 'brightness' },
  { label: 'Color Temperature', value: 'colorTemperature' },
  { label: 'Online Status', value: 'online' },
  { label: 'Custom (enter instance)', value: 'custom' },
];

// ---------------------------------------------------------------------------
// Query types
// ---------------------------------------------------------------------------

// The Govee API only returns point-in-time snapshots, not historical data.
// 'timeseries' is kept for backward-compatibility with existing saved queries
// but the UI no longer offers it as an option.
export type QueryType = 'timeseries' | 'current';

export const QUERY_TYPE_OPTIONS: Array<SelectableValue<QueryType>> = [
  { label: 'Current State', value: 'current' },
];

// ---------------------------------------------------------------------------
// Plugin query model
// ---------------------------------------------------------------------------

export interface GoveeQuery extends DataQuery {
  queryType: QueryType;
  /** Govee device MAC address / identifier */
  deviceId: string;
  /** Govee device SKU (model string) */
  sku: string;
  /** Capability instance to graph */
  metric: MetricInstance;
  /** Used when metric === 'custom' */
  customInstance?: string;
  /** Human-readable device name (display only) */
  deviceName?: string;
}

export const DEFAULT_QUERY: Partial<GoveeQuery> = {
  queryType: 'current',
  metric: 'temperature',
};

// ---------------------------------------------------------------------------
// Datasource config options
// ---------------------------------------------------------------------------

/**
 * Fields stored in jsonData (non-sensitive, persisted to DB).
 * We intentionally keep this minimal — the API key is in secureJsonData only.
 */
export interface GoveeDataSourceOptions extends DataSourceJsonData {
  /** Optional custom base URL override (useful for proxies / testing). */
  apiBaseUrl?: string;
}

/**
 * Fields stored in secureJsonData.
 * These are write-only from the frontend perspective — Grafana never sends
 * them back to the browser after they are saved.
 */
export interface GoveeSecureJsonData {
  apiKey?: string;
}

// ---------------------------------------------------------------------------
// CallResource response shapes
// ---------------------------------------------------------------------------

export interface DeviceListResponse {
  devices: GoveeDevice[];
}
