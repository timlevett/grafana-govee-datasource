// Minimal @grafana/runtime mock for Jest unit tests.

export class DataSourceWithBackend {
  url: string = '';
  constructor(_: any) {}

  // Defined on the prototype so that `super.query()` in DataSource works correctly.
  // Class-field syntax (query = jest.fn()) would set these only on instances,
  // making them unreachable via the `super` keyword.
  query(_req: any): Promise<any> {
    return Promise.resolve({ data: [] });
  }

  testDatasource(): Promise<any> {
    return Promise.resolve({ status: 'OK', message: 'Connected' });
  }
}

export const getBackendSrv = jest.fn().mockReturnValue({
  fetch: jest.fn().mockReturnValue({
    toPromise: jest.fn().mockResolvedValue({ data: { devices: [] } }),
  }),
});
