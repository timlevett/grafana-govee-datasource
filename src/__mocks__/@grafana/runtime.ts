// Minimal @grafana/runtime mock for Jest unit tests.

export class DataSourceWithBackend {
  url: string = '';
  constructor(_: any) {}
  query = jest.fn().mockResolvedValue({ data: [] });
  testDatasource = jest.fn().mockResolvedValue({ status: 'OK', message: 'Connected' });
}

export const getBackendSrv = jest.fn().mockReturnValue({
  fetch: jest.fn().mockReturnValue({
    toPromise: jest.fn().mockResolvedValue({ data: { devices: [] } }),
  }),
});
