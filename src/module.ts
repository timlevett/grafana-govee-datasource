import { DataSourcePlugin } from '@grafana/data';

import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { GoveeQuery, GoveeDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, GoveeQuery, GoveeDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
