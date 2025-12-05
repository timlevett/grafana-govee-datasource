import React, { ChangeEvent } from 'react';
import { InlineField, Input, Select, Stack } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

const queryTypes = [
  { label: 'List Devices', value: 'devices' },
  { label: 'Device State', value: 'deviceState' },
];

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onQueryTypeChange = (selected: { value?: string }) => {
    onChange({ ...query, queryType: selected.value || 'devices' });
    onRunQuery();
  };

  const onDeviceIdChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, deviceId: event.target.value });
  };

  const onModelChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, model: event.target.value });
  };

  const { queryType = 'devices', deviceId = '', model = '' } = query;

  return (
    <Stack gap={0}>
      <InlineField label="Query Type" labelWidth={14} tooltip="Select the type of query to execute">
        <Select
          id="query-editor-query-type"
          options={queryTypes}
          value={queryType}
          onChange={onQueryTypeChange}
          width={20}
        />
      </InlineField>
      {queryType === 'deviceState' && (
        <>
          <InlineField 
            label="Device ID" 
            labelWidth={14} 
            tooltip="The device ID from your Govee account"
            required
          >
            <Input
              id="query-editor-device-id"
              onChange={onDeviceIdChange}
              value={deviceId}
              placeholder="Enter device ID"
              width={30}
            />
          </InlineField>
          <InlineField 
            label="Model" 
            labelWidth={14} 
            tooltip="The device model (e.g., H6104)"
            required
          >
            <Input
              id="query-editor-model"
              onChange={onModelChange}
              value={model}
              placeholder="Enter device model"
              width={30}
            />
          </InlineField>
        </>
      )}
    </Stack>
  );
}
