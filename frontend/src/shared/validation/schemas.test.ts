import { describe, it, expect } from 'vitest';
import {
  passwordSchema,
  dotenvSchema,
  settingsRawSchema,
  formatZodIssues,
} from './schemas';

describe('passwordSchema', () => {
  it('accepts an 8-char password', () => {
    const result = passwordSchema.safeParse('password123');
    expect(result.success).toBe(true);
  });

  it('rejects a 7-char password', () => {
    const result = passwordSchema.safeParse('short1!');
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0]?.message).toContain('at least 8');
    }
  });

  it('rejects an empty string', () => {
    const result = passwordSchema.safeParse('');
    expect(result.success).toBe(false);
  });
});

describe('dotenvSchema', () => {
  it('accepts a typical .env file', () => {
    const content = [
      '# Comment line',
      'DB_HOST=localhost',
      'DB_PORT=5432',
      'MQTT_USER=admin',
      'MQTT_PASS=secret',
    ].join('\n');
    expect(dotenvSchema.safeParse(content).success).toBe(true);
  });

  it('accepts empty input', () => {
    expect(dotenvSchema.safeParse('').success).toBe(true);
  });

  it('rejects a malformed line (KEY with no =)', () => {
    const result = dotenvSchema.safeParse('NOEQUALS');
    expect(result.success).toBe(false);
  });

  it('rejects a leading-equals line', () => {
    const result = dotenvSchema.safeParse('=VALUE');
    expect(result.success).toBe(false);
  });

  it('rejects an empty key', () => {
    const result = dotenvSchema.safeParse('=VALUE');
    expect(result.success).toBe(false);
    if (!result.success) {
      const msg = result.error.issues.map((i) => i.message).join(' ');
      expect(msg).toMatch(/malformed|missing key/);
    }
  });

  it('accepts keys with dots', () => {
    const result = dotenvSchema.safeParse('app.config.value=42');
    expect(result.success).toBe(true);
  });
});

describe('settingsRawSchema', () => {
  it('accepts a simple module.exports', () => {
    const content = `module.exports = { uiPort: 1880 };`;
    expect(settingsRawSchema.safeParse(content).success).toBe(true);
  });

  it('accepts a typical settings.js with adminAuth', () => {
    const content = `module.exports = {
      uiPort: 1880,
      adminAuth: {
        type: 'credentials',
        users: [{ username: 'admin', password: 'hash', permissions: '*' }],
      },
    };`;
    expect(settingsRawSchema.safeParse(content).success).toBe(true);
  });

  it('rejects malformed JavaScript', () => {
    const result = settingsRawSchema.safeParse('module.exports = {');
    expect(result.success).toBe(false);
  });

  it('rejects unterminated string', () => {
    const result = settingsRawSchema.safeParse('module.exports = "open string');
    expect(result.success).toBe(false);
  });
});

describe('formatZodIssues', () => {
  it('produces a flat error map keyed by issue path', () => {
    const result = passwordSchema.safeParse('short');
    expect(result.success).toBe(false);
    if (!result.success) {
      const map = formatZodIssues(result.error);
      expect(Object.values(map).some((m) => m.includes('at least 8'))).toBe(true);
    }
  });
});
