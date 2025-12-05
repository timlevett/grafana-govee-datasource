import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { MyQuery, MyDataSourceOptions, DEFAULT_QUERY } from './types';

export class DataSource extends DataSourceWithBackend<MyQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<MyQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: MyQuery, scopedVars: ScopedVars) {
    return {
      ...query,
      deviceId: getTemplateSrv().replace(query.deviceId || '', scopedVars),
      model: getTemplateSrv().replace(query.model || '', scopedVars),
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
}
