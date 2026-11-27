import { describe, expect, it } from 'vitest';
import { DEFAULT_CONFIG, configToFormData, formDataToConfig } from './index';
import type { NodeRedConfig } from './index';

// Issue #1076 — the httpNodeCors block must survive the flat form model:
// `formDataToConfig(configToFormData(x))` preserves `x.httpNodeCors`.
const config = (httpNodeCors: NodeRedConfig['httpNodeCors']): NodeRedConfig => ({
  ...DEFAULT_CONFIG,
  httpNodeCors,
});

const roundTrip = (httpNodeCors: NodeRedConfig['httpNodeCors']) =>
  formDataToConfig(configToFormData(config(httpNodeCors))).httpNodeCors;

describe('httpNodeCors form mapping (issue #1076)', () => {
  it('round-trips an enabled block', () => {
    const cors = {
      origin: ['https://app.example.com'],
      methods: ['GET', 'POST'],
      credentials: true,
      allowedHeaders: ['Content-Type', 'Authorization'],
    };
    expect(roundTrip(cors)).toEqual(cors);
  });

  it('round-trips the wildcard origin on its own', () => {
    expect(roundTrip({ origin: ['*'] })).toEqual({ origin: ['*'] });
  });

  it('keeps the key absent when CORS is disabled', () => {
    const form = configToFormData(config(undefined));
    expect(form.httpNodeCorsEnabled).toBe(false);
    expect(formDataToConfig(form)).not.toHaveProperty('httpNodeCors');
  });

  it('round-trips an unsupported hand-written block unchanged', () => {
    const cors = {
      origin: [],
      unsupported: true,
      raw: '  httpNodeCors: {\n    origin: (req, cb) => cb(null, "*"),\n  },',
    };
    const form = configToFormData(config(cors));
    expect(form.httpNodeCorsUnsupported).toBe(true);
    expect(formDataToConfig(form).httpNodeCors).toEqual(cors);
  });

  it('defaults methods to the Node-RED default set and omits them again', () => {
    const form = configToFormData(config({ origin: ['*'] }));
    expect(form.httpNodeCorsMethods).toBe('GET,PUT,POST,DELETE');
    expect(formDataToConfig(form).httpNodeCors?.methods).toBeUndefined();
  });

  it('maps origins per line and ignores blank entries', () => {
    const form = {
      ...configToFormData(config(undefined)),
      httpNodeCorsEnabled: true,
      httpNodeCorsOrigins: 'https://a.example.com\n\n  \nhttps://b.example.com\n',
      httpNodeCorsMethods: 'GET, PUT',
      httpNodeCorsAllowedHeaders: 'Content-Type,',
    };
    expect(formDataToConfig(form).httpNodeCors).toEqual({
      origin: ['https://a.example.com', 'https://b.example.com'],
      methods: ['GET', 'PUT'],
      allowedHeaders: ['Content-Type'],
    });
  });
});
