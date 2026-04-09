import { test, expect } from '@grafana/plugin-e2e';

test('smoke: should render query editor with Device and Metric fields', async ({
  panelEditPage,
  readProvisionedDataSource,
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);
  const row = panelEditPage.getQueryEditorRow('A');
  await expect(row.getByText('Device')).toBeVisible();
  await expect(row.getByText('Metric')).toBeVisible();
});
