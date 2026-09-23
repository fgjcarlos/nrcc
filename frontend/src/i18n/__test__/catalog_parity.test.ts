import { describe, it, expect } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

// Catalog parity — every EN key must have a corresponding ES value.
// Prevents the truncation case where a translator adds a key in
// one locale but forgets the other. Slice 2 of issue #767.

const LOCALES_DIR = path.join(import.meta.dirname, '../../locales');
const NAMESPACES = ['common', 'auth', 'backups', 'configuration',
  'dashboard', 'env-vars', 'files', 'flows', 'libraries', 'updates'];

function flatten(obj: Record<string, unknown>, prefix = ''): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      Object.assign(out, flatten(v as Record<string, unknown>, key));
    } else {
      out[key] = String(v);
    }
  }
  return out;
}

describe('i18n catalog parity (slice 2 of #767)', () => {
  for (const ns of NAMESPACES) {
    const enPath = path.join(LOCALES_DIR, 'en', `${ns}.json`);
    const esPath = path.join(LOCALES_DIR, 'es', `${ns}.json`);
    if (!fs.existsSync(enPath) || !fs.existsSync(esPath)) {
      it.skip(`skipped: ${ns} (missing catalog)`, () => {});
      continue;
    }
    const en = flatten(JSON.parse(fs.readFileSync(enPath, 'utf-8')) as Record<string, unknown>);
    const es = flatten(JSON.parse(fs.readFileSync(esPath, 'utf-8')) as Record<string, unknown>);
    const enKeys = Object.keys(en).sort();
    const esKeys = Object.keys(es).sort();

    it(`${ns}: EN and ES keys are equal`, () => {
      expect(esKeys).toEqual(enKeys);
    });
    it(`${ns}: ES values are non-empty`, () => {
      for (const k of esKeys) {
        expect(es[k].trim(), `empty ES value at ${ns}:${k}`).not.toBe('');
      }
    });
  }
});
