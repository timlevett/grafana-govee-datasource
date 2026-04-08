import { DataSourceInstanceSettings, CoreApp, ScopedVars, MetricFindValue } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv, getBackendSrv } from '@grafana/runtime';

import { MyQuery, MyDataSourceOptions, DEFAULT_QUERY } from './types';

interface DeviceVariableOption {
  text: string;
  value: string;
}

interface DevicesResourceResponse {
  devices: DeviceVariableOption[];
  devicesWithModel: DeviceVariableOption[];
}

export class DataSource extends DataSourceWithBackend<MyQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<MyQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: MyQuery, scopedVars: ScopedVars) {
    let deviceId = getTemplateSrv().replace(query.deviceId || '', scopedVars);
    let model = getTemplateSrv().replace(query.model || '', scopedVars);

    // If deviceId contains the format "deviceId|model", parse it
    if (deviceId && deviceId.includes('|') && !model) {
      const parts = deviceId.split('|');
      if (parts.length === 2) {
        deviceId = parts[0];
        model = parts[1];
      }
    }

    return {
      ...query,
      deviceId,
      model,
    };
  }

  filterQuery(query: MyQuery): boolean {
    // For deviceState queries, require deviceId and model
    if (query.queryType === 'deviceState') {
      return !!(query.deviceId && query.model);
    }
    // For devices query, always allow
    return query.queryType === 'devices' || !!query.queryType;
  }

  /**
   * metricFindQuery is called by Grafana when a template variable queries this datasource.
   * Returns device list where:
   * - text (display): Device name
   * - value (passed): Device ID
   *
   * The query parameter can be used to specify which format to return:
   * - Empty or "devices": Returns device list with device name as text, device ID as value (default)
   * - "devicesWithModel": Returns device list with "Device Name (Model)" as text, "deviceId|model" as value
   */
  async metricFindQuery(query: string): Promise<MetricFindValue[]> {
    console.log('[metricFindQuery] Called with query:', query);

    // Default behavior: show device name, pass device ID
    // Only use model format if explicitly requested
    const useModel = query === 'devicesWithModel';
    const resourcePath = 'devices';

    console.log('[metricFindQuery] Calling getResource with path:', resourcePath);

    try {
      // Try using getResource (standard approach for DataSourceWithBackend)
      let data: DevicesResourceResponse;
      try {
        const response = await this.getResource<DevicesResourceResponse>(resourcePath);
        console.log('[metricFindQuery] getResource response:', response);
        data = response;
      } catch (getResourceError) {
        console.warn('[metricFindQuery] getResource failed, trying datasourceRequest:', getResourceError);
        // Fallback: use getBackendSrv directly
        const response = await getBackendSrv().datasourceRequest<DevicesResourceResponse>({
          url: `/api/datasources/${this.id}/resources/${resourcePath}`,
          method: 'GET',
        });
        console.log('[metricFindQuery] datasourceRequest response:', response);
        data = response.data;
      }

      if (!data) {
        console.error('Empty response from resource endpoint');
        return [];
      }

      // Select the appropriate array based on query
      // Default (devices): text = device name, value = device ID
      // With model: text = "Device Name (Model)", value = "deviceId|model"
      const options = useModel ? data.devicesWithModel : data.devices;

      if (!options || !Array.isArray(options)) {
        console.error('Invalid response format from resource endpoint:', data);
        return [];
      }

      // Convert to MetricFindValue format
      // text is what's displayed, value is what's passed to queries
      const result = options.map((option) => ({
        text: option.text || option.value || '', // Display: Device name (or "Device Name (Model)" if using model)
        value: option.value || option.text || '', // Value: Device ID (or "deviceId|model" if using model)
      }));

      console.log(`[metricFindQuery] Returning ${result.length} devices for query "${query}"`);
      return result;
    } catch (error) {
      console.error('[metricFindQuery] Error fetching devices for template variable:', error);
      console.error('[metricFindQuery] Error details:', {
        message: error instanceof Error ? error.message : String(error),
        stack: error instanceof Error ? error.stack : undefined,
      });
      // Return empty array on error (Grafana will show empty variable)
      return [];
    }
  }
}
