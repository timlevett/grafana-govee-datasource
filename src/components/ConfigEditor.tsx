import React, { ChangeEvent, PureComponent } from 'react';
import {
  DataSourcePluginOptionsEditorProps,
  updateDatasourcePluginJsonDataOption,
  updateDatasourcePluginSecureJsonDataOption,
} from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Field, Input, Button, Alert, SecretInput } from '@grafana/ui';

import { GoveeDataSourceOptions, GoveeSecureJsonData } from '../types';

type Props = DataSourcePluginOptionsEditorProps<GoveeDataSourceOptions, GoveeSecureJsonData>;

interface State {
  testing: boolean;
  testResult?: { ok: boolean; message: string };
}

export class ConfigEditor extends PureComponent<Props, State> {
  state: State = { testing: false };

  // -------------------------------------------------------------------------
  // API key (secureJsonData)
  // -------------------------------------------------------------------------

  onApiKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    updateDatasourcePluginSecureJsonDataOption(this.props, 'apiKey', event.target.value);
  };

  onResetApiKey = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: { apiKey: '' },
      secureJsonFields: { ...options.secureJsonFields, apiKey: false },
    });
  };

  // -------------------------------------------------------------------------
  // Optional base URL override
  // -------------------------------------------------------------------------

  onApiBaseUrlChange = (event: ChangeEvent<HTMLInputElement>) => {
    updateDatasourcePluginJsonDataOption(this.props, 'apiBaseUrl', event.target.value);
  };

  // -------------------------------------------------------------------------
  // Test connection
  // -------------------------------------------------------------------------

  onTest = async () => {
    const { options } = this.props;
    const uid = options.uid;

    if (!uid) {
      this.setState({ testResult: { ok: false, message: 'Save the datasource first, then test the connection.' } });
      return;
    }

    this.setState({ testing: true, testResult: undefined });
    try {
      const response = await getBackendSrv()
        .fetch<{ status: string; message: string }>({
          url: `/api/datasources/uid/${uid}/health`,
          method: 'GET',
        })
        .toPromise();

      const ok = response?.data?.status === 'OK';
      this.setState({
        testing: false,
        testResult: { ok, message: response?.data?.message ?? (ok ? 'Connection OK' : 'Health check failed') },
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      this.setState({ testing: false, testResult: { ok: false, message: `Connection failed: ${message}` } });
    }
  };

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  render() {
    const { options } = this.props;
    const { secureJsonFields, jsonData } = options;
    const { testing, testResult } = this.state;

    const isApiKeySet = Boolean(secureJsonFields?.apiKey);

    return (
      <div>
        {/* API Key */}
        <Field
          label="Govee API Key"
          description="Your Govee API key. Stored encrypted server-side — never sent back to the browser."
          required
        >
          <SecretInput
            width={40}
            id="govee-api-key"
            placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            isConfigured={isApiKeySet}
            onChange={this.onApiKeyChange}
            onReset={this.onResetApiKey}
          />
        </Field>

        {/* Optional base URL */}
        <Field
          label="API Base URL"
          description="Override the Govee API base URL. Leave empty to use the default: https://openapi.api.govee.com"
        >
          <Input
            width={40}
            id="govee-api-base-url"
            placeholder="https://openapi.api.govee.com"
            value={jsonData.apiBaseUrl ?? ''}
            onChange={this.onApiBaseUrlChange}
          />
        </Field>

        {/* Inline test result */}
        {testResult && (
          <Alert
            title={testResult.ok ? 'Connection OK' : 'Connection failed'}
            severity={testResult.ok ? 'success' : 'error'}
          >
            {testResult.message}
          </Alert>
        )}

        <Button
          variant="secondary"
          onClick={this.onTest}
          disabled={testing || !isApiKeySet}
          style={{ marginTop: '8px' }}
        >
          {testing ? 'Testing…' : 'Test connection'}
        </Button>

        <p style={{ marginTop: '16px', color: '#aaa', fontSize: '12px' }}>
          Govee API rate limit: 10,000 requests/day per account. The plugin tracks usage in-memory; restarts reset the
          counter. Obtain your API key from the Govee Home app: Profile → Settings → Apply for API Key.
        </p>
      </div>
    );
  }
}
