import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import { QueryEditor } from '../components/QueryEditor';
import { GoveeDevice, GoveeQuery, METRIC_OPTIONS } from '../types';

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

function makeDevice(overrides: Partial<GoveeDevice> = {}): GoveeDevice {
  return {
    sku: 'H5101',
    device: 'AA:BB:CC',
    deviceName: 'Living Room Sensor',
    type: 'sensor',
    capabilities: [
      { type: 'devices.capabilities.property', instance: 'temperature' },
      { type: 'devices.capabilities.property', instance: 'humidity' },
    ],
    ...overrides,
  };
}

function makeQuery(overrides: Partial<GoveeQuery> = {}): GoveeQuery {
  return {
    refId: 'A',
    queryType: 'current',
    deviceId: '',
    sku: '',
    metric: 'temperature',
    ...overrides,
  } as any;
}

function makeDatasource(devices: GoveeDevice[] = []) {
  return {
    getDevices: jest.fn().mockResolvedValue(devices),
    url: '/api/datasources/proxy/1',
  } as any;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function renderEditor(props: Partial<Parameters<typeof QueryEditor>[0]> = {}) {
  const defaults = {
    query: makeQuery(),
    onChange: jest.fn(),
    onRunQuery: jest.fn(),
    datasource: makeDatasource(),
  };
  return render(<QueryEditor {...defaults} {...(props as any)} />);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('QueryEditor', () => {
  it('renders without crashing', () => {
    renderEditor();
  });

  it('calls datasource.getDevices on mount', async () => {
    const datasource = makeDatasource();
    renderEditor({ datasource });
    await waitFor(() => {
      expect(datasource.getDevices).toHaveBeenCalledTimes(1);
    });
  });

  it('shows help text when no device is selected', () => {
    renderEditor({ query: makeQuery({ deviceId: '' }) });
    expect(screen.getByText(/Select a device to begin/i)).toBeTruthy();
  });

  it('does not show help text when a device is selected', async () => {
    renderEditor({ query: makeQuery({ deviceId: 'AA:BB:CC' }) });
    expect(screen.queryByText(/Select a device to begin/i)).toBeNull();
  });

  it('shows custom instance input when metric is "custom"', () => {
    renderEditor({ query: makeQuery({ metric: 'custom' }) });
    expect(screen.getByPlaceholderText(/e\.g\. sensorTemperature/i)).toBeTruthy();
  });

  it('hides custom instance input when metric is not "custom"', () => {
    renderEditor({ query: makeQuery({ metric: 'temperature' }) });
    expect(screen.queryByPlaceholderText(/e\.g\. sensorTemperature/i)).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// getMetricOptions logic
// ---------------------------------------------------------------------------

describe('QueryEditor.getMetricOptions', () => {
  // Exercise the filtering logic by controlling the device list in state.
  // We render the component with a device set, then verify the select options
  // by inspecting the component via a custom subclass.

  it('returns all METRIC_OPTIONS when no device is selected', async () => {
    // Instantiate via render and call the method via a test ref.
    let instance: InstanceType<typeof QueryEditor> | null = null;
    const RefCapture = class extends QueryEditor {
      constructor(props: any) {
        super(props);
        instance = this;
      }
    };

    const ds = makeDatasource([]);
    render(
      <RefCapture
        query={makeQuery({ deviceId: '' })}
        onChange={jest.fn()}
        onRunQuery={jest.fn()}
        datasource={ds}
      />
    );

    await waitFor(() => expect(ds.getDevices).toHaveBeenCalled());

    expect(instance).not.toBeNull();
    const opts = (instance as any).getMetricOptions();
    expect(opts).toEqual(METRIC_OPTIONS);
  });

  it('filters metric options to device capabilities when device is selected', async () => {
    const device = makeDevice(); // has temperature, humidity capabilities
    const ds = makeDatasource([device]);

    let instance: InstanceType<typeof QueryEditor> | null = null;
    const RefCapture = class extends QueryEditor {
      constructor(props: any) {
        super(props);
        instance = this;
      }
    };

    await act(async () => {
      render(
        <RefCapture
          query={makeQuery({ deviceId: device.device })}
          onChange={jest.fn()}
          onRunQuery={jest.fn()}
          datasource={ds}
        />
      );
    });

    await waitFor(() => expect(ds.getDevices).toHaveBeenCalled());

    const opts = (instance as any).getMetricOptions() as typeof METRIC_OPTIONS;
    const values = opts.map((o: any) => o.value);
    expect(values).toContain('temperature');
    expect(values).toContain('humidity');
    // Options not in device capabilities are filtered out
    expect(values).not.toContain('battery');
    // Custom is always present
    expect(values).toContain('custom');
  });

  it('always includes "custom" option', async () => {
    const device = makeDevice({
      capabilities: [{ type: 'test', instance: 'temperature' }],
    });
    const ds = makeDatasource([device]);

    let instance: InstanceType<typeof QueryEditor> | null = null;
    const RefCapture = class extends QueryEditor {
      constructor(props: any) {
        super(props);
        instance = this;
      }
    };

    await act(async () => {
      render(
        <RefCapture
          query={makeQuery({ deviceId: device.device })}
          onChange={jest.fn()}
          onRunQuery={jest.fn()}
          datasource={ds}
        />
      );
    });

    await waitFor(() => expect(ds.getDevices).toHaveBeenCalled());

    const opts = (instance as any).getMetricOptions();
    const customOpt = opts.find((o: any) => o.value === 'custom');
    expect(customOpt).toBeDefined();
  });
});

// ---------------------------------------------------------------------------
// getDeviceOptions logic
// ---------------------------------------------------------------------------

describe('QueryEditor.getDeviceOptions', () => {
  it('maps devices to selectable values with correct label and description', async () => {
    const device = makeDevice();
    const ds = makeDatasource([device]);

    let instance: InstanceType<typeof QueryEditor> | null = null;
    const RefCapture = class extends QueryEditor {
      constructor(props: any) {
        super(props);
        instance = this;
      }
    };

    await act(async () => {
      render(
        <RefCapture
          query={makeQuery()}
          onChange={jest.fn()}
          onRunQuery={jest.fn()}
          datasource={ds}
        />
      );
    });

    await waitFor(() => expect(ds.getDevices).toHaveBeenCalled());

    const opts = (instance as any).getDeviceOptions();
    expect(opts).toHaveLength(1);
    expect(opts[0].value).toBe(device.device);
    expect(opts[0].label).toBe(device.deviceName);
    expect(opts[0].description).toContain(device.sku);
  });
});
