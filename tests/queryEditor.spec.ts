import { test, expect } from '@grafana/plugin-e2e';

/**
 * Query editor e2e tests.
 *
 * These tests require GOVEE_API_KEY to be set in the environment and a running
 * Grafana instance (via docker compose) with the plugin loaded. They are skipped
 * automatically when the environment variable is absent.
 *
 * Run with:
 *   GOVEE_API_KEY=<your-key> npm run e2e
 */

const API_KEY = process.env.GOVEE_API_KEY ?? '';

test.describe('Query editor', () => {
  test.skip(!API_KEY, 'GOVEE_API_KEY not set — skipping live query tests');

  test('device dropdown populates when API key is configured', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    panelEditPage,
    page,
  }) => {
    // Create a datasource with a real API key.
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    const configPage = await createDataSourceConfigPage({ type: ds.type });
    await page.getByRole('textbox', { name: /API key/i }).fill(API_KEY);
    await expect(configPage.saveAndTest()).toBeOK();

    // Open a new panel using the configured datasource.
    await panelEditPage.datasource.set(ds.name);

    // The device dropdown should load and show at least one option.
    const deviceSelect = page.getByRole('combobox', { name: /Device/i });
    await deviceSelect.click();

    // Wait for devices to load (loading spinner disappears).
    await expect(page.getByText('Loading devices')).not.toBeVisible({ timeout: 10_000 });

    // The "No devices found" message should not appear.
    await expect(page.getByText('No devices found. Check your API key.')).not.toBeVisible();
  });

  test('selecting a device and brightness metric returns data without error', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    panelEditPage,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    const configPage = await createDataSourceConfigPage({ type: ds.type });
    await page.getByRole('textbox', { name: /API key/i }).fill(API_KEY);
    await expect(configPage.saveAndTest()).toBeOK();

    await panelEditPage.datasource.set(ds.name);

    // Open device dropdown and pick the first available device.
    const deviceSelect = page.getByRole('combobox', { name: /Device/i });
    await deviceSelect.click();
    await expect(page.getByText('Loading devices')).not.toBeVisible({ timeout: 10_000 });

    const firstOption = page.locator('[class*="menu"] [class*="option"]').first();
    await firstOption.waitFor({ state: 'visible', timeout: 10_000 });
    await firstOption.click();

    // Select "Brightness" as the metric — all lights have this capability.
    const metricSelect = page.getByRole('combobox', { name: /Metric/i });
    await metricSelect.click();
    await page.getByText('Brightness', { exact: true }).click();

    // Run the query and verify it returns a result without an error panel.
    await panelEditPage.refreshPanel();

    // Grafana renders an error notice when the backend returns an error response.
    await expect(page.locator('[data-testid="data-testid Panel status error"]')).not.toBeVisible({
      timeout: 15_000,
    });
  });
});
