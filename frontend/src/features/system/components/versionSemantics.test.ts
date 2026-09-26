import { describe, expect, it } from 'vitest';
import { getVersionVariant, parseMajorVersion } from './versionSemantics';

describe('getVersionVariant', () => {
  it('returns success for Node-RED 5.x', () => {
    expect(getVersionVariant('5.0.7')).toBe('success');
    expect(getVersionVariant('5.0.0')).toBe('success');
    expect(getVersionVariant('5.4.2')).toBe('success');
    expect(getVersionVariant('5.99.0')).toBe('success');
  });

  it('returns warning for Node-RED 4.x', () => {
    expect(getVersionVariant('4.0.2')).toBe('warning');
    expect(getVersionVariant('4.0.9')).toBe('warning');
    expect(getVersionVariant('4.99.0')).toBe('warning');
  });

  it('returns danger for versions below 4.x', () => {
    expect(getVersionVariant('3.0.0')).toBe('danger');
    expect(getVersionVariant('2.1.5')).toBe('danger');
    expect(getVersionVariant('1.0.0')).toBe('danger');
  });

  it('returns danger for parse failures', () => {
    expect(getVersionVariant('unknown')).toBe('danger');
    expect(getVersionVariant('garbage')).toBe('danger');
    expect(getVersionVariant('not-a-version')).toBe('danger');
  });

  it('returns neutral for missing versions', () => {
    expect(getVersionVariant(undefined)).toBe('neutral');
    expect(getVersionVariant(null)).toBe('neutral');
    expect(getVersionVariant('')).toBe('neutral');
  });

  it('accepts the leading-v prefix', () => {
    expect(getVersionVariant('v5.0.7')).toBe('success');
    expect(getVersionVariant('v4.0.0')).toBe('warning');
  });

  it('trims whitespace', () => {
    expect(getVersionVariant('  5.0.7  ')).toBe('success');
    expect(getVersionVariant('\t4.0.2\n')).toBe('warning');
  });
});

describe('parseMajorVersion', () => {
  it('extracts the major number from common formats', () => {
    expect(parseMajorVersion('5.0.7')).toBe(5);
    expect(parseMajorVersion('5')).toBe(5);
    expect(parseMajorVersion('v5.0.7')).toBe(5);
    expect(parseMajorVersion('4.0.0-beta')).toBe(4);
  });

  it('returns null for malformed input', () => {
    expect(parseMajorVersion('unknown')).toBeNull();
    expect(parseMajorVersion(undefined)).toBeNull();
    expect(parseMajorVersion(null)).toBeNull();
    expect(parseMajorVersion('')).toBeNull();
  });
});
