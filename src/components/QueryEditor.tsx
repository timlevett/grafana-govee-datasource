import React, { PureComponent } from 'react';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { Select, Input, InlineFieldRow, InlineField } from '@grafana/ui';

import { DataSource } from '../datasource';
import {
  GoveeDataSourceOptions,
  GoveeQuery,
  GoveeDevice,
  MetricInstance,
  METRIC_OPTIONS,
  QueryType,
  QUERY_TYPE_OPTIONS,
} from '../types';

type Props = QueryEditorProps<DataSource, GoveeQuery, GoveeDataSourceOptions>;

interface State {
  devices: GoveeDevice[];
  devicesLoading: boolean;
}

export class QueryEditor extends PureComponent<Props, State> {
  state: State = {
    devices: [],
    devicesLoading: false,
  };

  componentDidMount() {
    this.loadDevices();
  }

  // -------------------------------------------------------------------------
  // Device loading
  // -------------------------------------------------------------------------

  loadDevices = async () => {
    this.setState({ devicesLoading: true });
    try {
      const devices = await this.props.datasource.getDevices();
      this.setState({ devices, devicesLoading: false });
    } catch {
      this.setState({ devicesLoading: false });
    }
  };

  getDeviceOptions = (): Array<SelectableValue<string>> => {
    return this.state.devices.map((d) => ({
      label: d.deviceName || d.device,
      value: d.device,
      description: `${d.sku} — ${d.type}`,
    }));
  };

  /**
   * Returns capability instances available on the currently selected device.
   * Falls back to the full METRIC_OPTIONS list when no device is selected.
   */
  getMetricOptions = (): Array<SelectableValue<MetricInstance>> => {
    const { query } = this.props;
    const { devices } = this.state;
    const selectedDevice = devices.find((d) => d.device === query.deviceId);

    if (!selectedDevice) {
      return METRIC_OPTIONS;
    }

    const availableInstances = new Set(selectedDevice.capabilities.map((c) => c.instance));
    const filtered = METRIC_OPTIONS.filter(
      (opt) => opt.value === 'custom' || (opt.value && availableInstances.has(opt.value))
    );

    // Always include "custom" at the end
    if (!filtered.find((o) => o.value === 'custom')) {
      filtered.push(METRIC_OPTIONS.find((o) => o.value === 'custom')!);
    }

    return filtered.length > 1 ? filtered : METRIC_OPTIONS;
  };

  // -------------------------------------------------------------------------
  // Change handlers
  // -------------------------------------------------------------------------

  onQueryTypeChange = (value: SelectableValue<QueryType>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, queryType: value.value! });
    onRunQuery();
  };

  onDeviceChange = (value: SelectableValue<string>) => {
    const { onChange, query, onRunQuery } = this.props;
    const device = this.state.devices.find((d) => d.device === value.value);
    onChange({
      ...query,
      deviceId: value.value ?? '',
      sku: device?.sku ?? '',
      deviceName: device?.deviceName ?? value.label ?? '',
    });
    onRunQuery();
  };

  onMetricChange = (value: SelectableValue<MetricInstance>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, metric: value.value! });
    onRunQuery();
  };

  onCustomInstanceChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, customInstance: event.target.value });
  };

  onCustomInstanceBlur = () => {
    this.props.onRunQuery();
  };

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  render() {
    const { query } = this.props;
    const { devicesLoading } = this.state;

    const selectedDevice = query.deviceId
      ? { label: query.deviceName || query.deviceId, value: query.deviceId }
      : undefined;

    const selectedMetric = METRIC_OPTIONS.find((o) => o.value === query.metric) ?? METRIC_OPTIONS[0];
    const selectedQueryType = QUERY_TYPE_OPTIONS.find((o) => o.value === query.queryType) ?? QUERY_TYPE_OPTIONS[1];

    return (
      <div>
        <InlineFieldRow>
          {/* Query type */}
          <InlineField label="Query Type" labelWidth={14}>
            <Select
              width={20}
              options={QUERY_TYPE_OPTIONS}
              value={selectedQueryType}
              onChange={this.onQueryTypeChange}
            />
          </InlineField>

          {/* Device selector */}
          <InlineField label="Device" labelWidth={10} grow>
            <Select
              width={32}
              placeholder={devicesLoading ? 'Loading devices…' : 'Select device'}
              options={this.getDeviceOptions()}
              value={selectedDevice}
              onChange={this.onDeviceChange}
              isLoading={devicesLoading}
              noOptionsMessage="No devices found. Check your API key."
            />
          </InlineField>

          {/* Refresh devices button */}
          <InlineField label="" labelWidth={0}>
            <button
              className="btn btn-secondary btn-small"
              onClick={this.loadDevices}
              title="Refresh device list"
              style={{ marginLeft: '4px' }}
            >
              ↺
            </button>
          </InlineField>
        </InlineFieldRow>

        <InlineFieldRow>
          {/* Metric / capability */}
          <InlineField
            label="Metric"
            labelWidth={14}
            tooltip="The capability instance to query from the device state."
          >
            <Select
              width={24}
              options={this.getMetricOptions()}
              value={selectedMetric}
              onChange={this.onMetricChange}
            />
          </InlineField>

          {/* Custom instance input */}
          {query.metric === 'custom' && (
            <InlineField label="Instance name" labelWidth={16}>
              <Input
                width={20}
                placeholder="e.g. sensorTemperature"
                value={query.customInstance ?? ''}
                onChange={this.onCustomInstanceChange}
                onBlur={this.onCustomInstanceBlur}
              />
            </InlineField>
          )}
        </InlineFieldRow>

        {!query.deviceId && (
          <div style={{ color: '#888', fontSize: '12px', marginTop: '4px' }}>
            Select a device to begin. Devices are fetched from your Govee account.
          </div>
        )}
      </div>
    );
  }
}
