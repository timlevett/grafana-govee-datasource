import {
  DataSourceInstanceSettings,
  CoreApp,
  ScopedVars,
  DataQueryRequest,
  DataQueryResponse,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { of, Observable } from 'rxjs';

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

  query(request: DataQueryRequest<GoveeQuery>): Observable<DataQueryResponse> {
    // Filter out targets with no device selected
    const targets = request.targets.filter((t) => !t.hide && t.deviceId && t.sku);
    if (targets.length === 0) {
      return of({ data: [] });
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
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      deviceId: templateSrv.replace(query.deviceId ?? '', scopedVars),
      metric: templateSrv.replace(query.metric ?? '', scopedVars) as GoveeQuery['metric'],
      customInstance: query.customInstance
        ? templateSrv.replace(query.customInstance, scopedVars)
        : query.customInstance,
    };
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
      const response = await this.getResource<DeviceListResponse>('devices');
      return response?.devices ?? [];
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error('GoveeDataSource: getDevices failed', message);
      return [];
    }
  }
}
