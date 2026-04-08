import {
  DataSourceInstanceSettings,
  CoreApp,
  ScopedVars,
  DataQueryRequest,
} from '@grafana/data';
import { DataSourceWithBackend, getBackendSrv } from '@grafana/runtime';

import {
  GoveeQuery,
  GoveeDataSourceOptions,
  DEFAULT_QUERY,
  GoveeDevice,
  DeviceListResponse,
} from './types';

export class DataSource extends DataSourceWithBackend<GoveeQuery, GoveeDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<GoveeDataSourceOptions>) {
    super(instanceSettings);
  }

  // -------------------------------------------------------------------------
  // Query — delegated to backend via DataSourceWithBackend
  // -------------------------------------------------------------------------

  query(request: DataQueryRequest<GoveeQuery>): ReturnType<DataSourceWithBackend<GoveeQuery, GoveeDataSourceOptions>['query']> {
    // Filter out targets with no device selected
    const targets = request.targets.filter((t) => !t.hide && t.deviceId && t.sku);
    if (targets.length === 0) {
      return Promise.resolve({ data: [] });
    }
    return super.query({ ...request, targets });
  }

  // -------------------------------------------------------------------------
  // Apply default values for new queries
  // -------------------------------------------------------------------------

  getDefaultQuery(_app: CoreApp): Partial<GoveeQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: GoveeQuery, scopedVars: ScopedVars): GoveeQuery {
    return query;
  }

  // -------------------------------------------------------------------------
  // Health check — delegates to backend CheckHealth
  // -------------------------------------------------------------------------

  async testDatasource(): Promise<{ status: string; message: string }> {
    return super.testDatasource();
  }

  // -------------------------------------------------------------------------
  // Resource calls (CallResource on the backend)
  // -------------------------------------------------------------------------

  /**
   * Fetch the list of Govee devices associated with the configured API key.
   * The API key is never exposed to the browser — it lives only in the backend.
   */
  async getDevices(): Promise<GoveeDevice[]> {
    try {
      const response = await getBackendSrv().fetch<DeviceListResponse>({
        url: `${this.url}/resources/devices`,
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      }).toPromise();

      return response?.data?.devices ?? [];
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error('GoveeDataSource: getDevices failed', message);
      return [];
    }
  }
}
