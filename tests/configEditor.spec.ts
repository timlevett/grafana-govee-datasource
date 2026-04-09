import { test, expect } from '@grafana/plugin-e2e';

test('smoke: should render config editor with API key field', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await createDataSourceConfigPage({ type: ds.type });
  // The field label renders as "Govee API Key *" — use a more specific selector
  await expect(page.getByText('Govee API Key *')).toBeVisible();
});

test('"Save & test" should fail when API key is not configured', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  const configPage = await createDataSourceConfigPage({ type: ds.type });
  // Do not fill in API key — backend should reject with "API key is not configured"
  await expect(configPage.saveAndTest()).not.toBeOK();
  await expect(configPage).toHaveAlert('error', { hasText: 'API key is not configured' });
});
