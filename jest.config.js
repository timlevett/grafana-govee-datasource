/** @type {import('jest').Config} */
module.exports = {
  // Use jsdom to simulate a browser environment for React component tests.
  testEnvironment: 'jest-environment-jsdom',

  // Transform TypeScript and TSX with SWC for fast test runs.
  transform: {
    '^.+\\.[tj]sx?$': [
      '@swc/jest',
      {
        jsc: {
          target: 'es2018',
          parser: {
            syntax: 'typescript',
            tsx: true,
            decorators: false,
          },
        },
      },
    ],
  },

  // Map Grafana UI imports to identity-obj-proxy for isolated unit tests.
  moduleNameMapper: {
    // CSS modules
    '\\.css$': 'identity-obj-proxy',
    '\\.scss$': 'identity-obj-proxy',
    '\\.sass$': 'identity-obj-proxy',

    // SVG / image assets
    '\\.(png|jpg|jpeg|gif|svg|eot|ttf|woff|woff2)$': '<rootDir>/src/__mocks__/fileMock.js',

    // Grafana externals — map to mocks so unit tests don't pull in full Grafana.
    '@grafana/data': '<rootDir>/src/__mocks__/@grafana/data.ts',
    '@grafana/runtime': '<rootDir>/src/__mocks__/@grafana/runtime.ts',
    '@grafana/ui': '<rootDir>/src/__mocks__/@grafana/ui.ts',
  },

  // Only run files in src/
  roots: ['<rootDir>/src'],

  // Test file patterns
  testMatch: ['**/__tests__/**/*.{ts,tsx}', '**/*.{spec,test}.{ts,tsx}'],

  // React 18 + @testing-library/react requires IS_REACT_ACT_ENVIRONMENT=true
  // so that act() warnings from async state updates are surfaced correctly.
  globals: {
    IS_REACT_ACT_ENVIRONMENT: true,
  },

  // Run before the test framework is installed — sets NODE_ENV=test so React
  // loads its development build (which enables act() support).
  setupFiles: ['<rootDir>/jest.setup.js'],

  // Runs after the test framework is set up — use for jest-dom matchers etc.
  // This key is correct for Jest 27+.
  setupFilesAfterEnv: ['@testing-library/jest-dom'],

  // Collect coverage from src/ only, excluding mocks and type-only files.
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/__mocks__/**',
    '!src/**/__tests__/**',
    '!src/types.ts',
    '!src/module.ts',
  ],
};
