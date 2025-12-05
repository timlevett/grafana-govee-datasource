import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  // Secure field (only sent to the backend)
  const onAPIKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        apiKey: event.target.value,
      },
    });
  };

  const onResetAPIKey = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiKey: '',
      },
    });
  };

  return (
    <>
      <InlineField 
        label="Govee API Key" 
        labelWidth={14} 
        interactive 
        tooltip="Your Govee API key. Get it from the Govee Home app: My Profile > Settings > Apply for API Key"
      >
        <SecretInput
          required
          id="config-editor-api-key"
          isConfigured={secureJsonFields.apiKey}
          value={secureJsonData?.apiKey}
          placeholder="Enter your Govee API key"
          width={40}
          onReset={onResetAPIKey}
          onChange={onAPIKeyChange}
        />
      </InlineField>
    </>
  );
}
