// @ts-check
/** @type {import('eslint').Linter.Config} */
module.exports = {
  extends: ['@grafana/eslint-config'],
  root: true,
  env: {
    browser: true,
    node: true,
  },
  rules: {
    // Disallow console.log in production code — use @grafana/runtime's AppEvents
    // or structured logging. We allow console.error/warn for now.
    'no-console': ['error', { allow: ['error', 'warn'] }],

    // Enforce explicit return types on exported functions for Go-like clarity.
    '@typescript-eslint/explicit-module-boundary-types': 'warn',

    // Govee API key must never appear in template literals or string concat.
    // (This is a belt-and-suspenders rule; the real guard is secureJsonData.)
    'no-restricted-syntax': [
      'error',
      {
        selector: 'Identifier[name="apiKey"]',
        message:
          'Do not reference apiKey in frontend code. The API key must only live in secureJsonData and the Go backend.',
      },
    ],
  },
  overrides: [
    {
      // Relax some rules for config and mock files
      files: ['webpack.config.ts', 'jest.config.js', '.eslintrc.js', 'src/__mocks__/**/*.ts', 'src/__mocks__/**/*.tsx'],
      rules: {
        '@typescript-eslint/explicit-module-boundary-types': 'off',
        '@typescript-eslint/no-inferrable-types': 'off',
        'react/display-name': 'off',
        'no-restricted-syntax': 'off',
      },
    },
    {
      // Relax console in types.ts (it's pure data)
      files: ['src/types.ts'],
      rules: {
        'no-restricted-syntax': 'off',
      },
    },
    {
      // ConfigEditor legitimately checks secureJsonFields.apiKey (a boolean
      // "is-configured" flag — the actual key value is never accessible in the
      // browser). Allow the identifier in this file only.
      files: ['src/components/ConfigEditor.tsx'],
      rules: {
        'no-restricted-syntax': 'off',
      },
    },
    {
      // Test files may reference field names (apiKey) when filling in forms
      // via provisioned datasource data. The key never leaves the test context.
      files: ['tests/**/*.ts', 'tests/**/*.tsx'],
      rules: {
        'no-restricted-syntax': 'off',
      },
    },
  ],
};
