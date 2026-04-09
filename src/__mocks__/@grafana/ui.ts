// Minimal @grafana/ui mock for Jest unit tests.
// We render plain HTML elements so tests don't pull in the full Grafana design system.

import React from 'react';

const stub =
  (name: string) =>
  ({ children, ...props }: any) =>
    React.createElement(name, props, children);

export const Field = stub('div');
export const Input = stub('input');
export const Button = stub('button');
export const Alert = stub('div');
export const SecretInput = stub('input');
export const Select = stub('select');
export const InlineFieldRow = stub('div');
export const InlineField = stub('div');
export const AsyncSelect = stub('select');
