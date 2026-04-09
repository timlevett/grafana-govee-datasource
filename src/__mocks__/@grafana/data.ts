// Minimal @grafana/data mock for Jest unit tests.
// Only exports used by the plugin source are stubbed out.

export const DataSourcePlugin = jest.fn().mockImplementation(() => ({
  setConfigEditor: jest.fn().mockReturnThis(),
  setQueryEditor: jest.fn().mockReturnThis(),
}));

export const updateDatasourcePluginJsonDataOption = jest.fn();
export const updateDatasourcePluginSecureJsonDataOption = jest.fn();

export interface DataSourceJsonData {}
export interface DataQueryRequest<T = any> {
  targets: T[];
  range?: any;
}
export interface DataQueryResponse {
  data: any[];
}
export interface SelectableValue<T = any> {
  label?: string;
  value?: T;
  description?: string;
}

export const CoreApp = { Dashboard: 'dashboard' };
