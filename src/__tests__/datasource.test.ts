import { DataSource } from '../datasource';
import { DataSourceWithBackend, getBackendSrv } from '@grafana/runtime';
import { GoveeQuery } from '../types';

// Helper to build a minimal DataSource instance for tests.
function makeDS(): DataSource {
  return new DataSource({ name: 'test', uid: 'test', url: '/api/proxy/1', type: 'govee' } as any);
}

// Helper to build a minimal GoveeQuery.
function makeQuery(overrides: Partial<GoveeQuery> = {}): GoveeQuery {
  return {
    refId: 'A',
    queryType: 'current',
    deviceId: 'AA:BB:CC',
    sku: 'H5101',
    metric: 'temperature',
    ...overrides,
  } as any;
}

// ---------------------------------------------------------------------------
// query() — filtering logic
// ---------------------------------------------------------------------------

describe('DataSource.query', () => {
  let ds: DataSource;
  let superQuerySpy: jest.SpyInstance;

  beforeEach(() => {
    ds = makeDS();
    superQuerySpy = jest
      .spyOn(DataSourceWithBackend.prototype, 'query')
      .mockResolvedValue({ data: ['frame'] } as any);
  });

  afterEach(() => {
    superQuerySpy.mockRestore();
  });

  it('returns empty data immediately when all targets are hidden', async () => {
    const request: any = { targets: [makeQuery({ hide: true })] };
    const result = await ds.query(request);
    expect(result).toEqual({ data: [] });
    expect(superQuerySpy).not.toHaveBeenCalled();
  });

  it('returns empty data when all targets have no deviceId', async () => {
    const request: any = { targets: [makeQuery({ deviceId: '' })] };
    const result = await ds.query(request);
    expect(result).toEqual({ data: [] });
    expect(superQuerySpy).not.toHaveBeenCalled();
  });

  it('returns empty data when all targets have no sku', async () => {
    const request: any = { targets: [makeQuery({ sku: '' })] };
    const result = await ds.query(request);
    expect(result).toEqual({ data: [] });
    expect(superQuerySpy).not.toHaveBeenCalled();
  });

  it('returns empty data when targets array is empty', async () => {
    const request: any = { targets: [] };
    const result = await ds.query(request);
    expect(result).toEqual({ data: [] });
    expect(superQuerySpy).not.toHaveBeenCalled();
  });

  it('calls super.query with valid targets', async () => {
    const validTarget = makeQuery();
    const request: any = { targets: [validTarget] };
    const result = await ds.query(request);
    expect(superQuerySpy).toHaveBeenCalledTimes(1);
    const passedRequest = superQuerySpy.mock.calls[0][0];
    expect(passedRequest.targets).toHaveLength(1);
    expect(passedRequest.targets[0].deviceId).toBe('AA:BB:CC');
    expect(result.data).toEqual(['frame']);
  });

  it('filters hidden targets and forwards remaining to super.query', async () => {
    const hiddenTarget = makeQuery({ refId: 'A', hide: true, deviceId: 'XX:YY' });
    const visibleTarget = makeQuery({ refId: 'B', deviceId: 'AA:BB:CC' });
    const request: any = { targets: [hiddenTarget, visibleTarget] };
    await ds.query(request);
    const passedRequest = superQuerySpy.mock.calls[0][0];
    expect(passedRequest.targets).toHaveLength(1);
    expect(passedRequest.targets[0].deviceId).toBe('AA:BB:CC');
  });

  it('filters targets with missing deviceId and forwards remaining', async () => {
    const noDevice = makeQuery({ refId: 'A', deviceId: '' });
    const valid = makeQuery({ refId: 'B', deviceId: 'AA:BB:CC' });
    const request: any = { targets: [noDevice, valid] };
    await ds.query(request);
    const passedRequest = superQuerySpy.mock.calls[0][0];
    expect(passedRequest.targets).toHaveLength(1);
    expect(passedRequest.targets[0].refId).toBe('B');
  });
});

// ---------------------------------------------------------------------------
// getDevices() — resource call
// ---------------------------------------------------------------------------

describe('DataSource.getDevices', () => {
  let ds: DataSource;

  beforeEach(() => {
    ds = makeDS();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('returns devices from backend resource endpoint', async () => {
    const mockDevices = [{ sku: 'H5101', device: 'AA:BB', deviceName: 'Sensor', type: 'sensor', capabilities: [] }];
    const fetchMock = jest.fn().mockReturnValue({
      toPromise: jest.fn().mockResolvedValue({ data: { devices: mockDevices } }),
    });
    (getBackendSrv as jest.Mock).mockReturnValue({ fetch: fetchMock });

    const result = await ds.getDevices();
    expect(result).toEqual(mockDevices);
    expect(fetchMock).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET' })
    );
  });

  it('returns empty array when response has no devices', async () => {
    const fetchMock = jest.fn().mockReturnValue({
      toPromise: jest.fn().mockResolvedValue({ data: {} }),
    });
    (getBackendSrv as jest.Mock).mockReturnValue({ fetch: fetchMock });

    const result = await ds.getDevices();
    expect(result).toEqual([]);
  });

  it('returns empty array when fetch throws', async () => {
    const fetchMock = jest.fn().mockReturnValue({
      toPromise: jest.fn().mockRejectedValue(new Error('network error')),
    });
    (getBackendSrv as jest.Mock).mockReturnValue({ fetch: fetchMock });

    const result = await ds.getDevices();
    expect(result).toEqual([]);
  });
});
