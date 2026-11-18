import { describe, expect, it } from 'vitest';
import { contrastRatio, meetsAA, parseColor } from './contrast';

describe('parseColor', () => {
  it('reads 6- and 3-digit hex and rgb() in both syntaxes', () => {
    expect(parseColor('#2b5fb4')).toEqual([43, 95, 180]);
    expect(parseColor(' #FFF ')).toEqual([255, 255, 255]);
    expect(parseColor('rgb(21, 23, 28)')).toEqual([21, 23, 28]);
    expect(parseColor('rgb(21 23 28 / 0.08)')).toEqual([21, 23, 28]);
  });

  it('returns null for anything it cannot measure', () => {
    expect(parseColor('')).toBeNull();
    expect(parseColor('var(--x)')).toBeNull();
    expect(parseColor('#12345')).toBeNull();
  });
});

describe('contrastRatio', () => {
  it('matches the WCAG reference values', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 5);
    expect(contrastRatio('#ffffff', '#ffffff')).toBeCloseTo(1, 5);
    // #767676 is the classic lightest grey that clears 4.5:1 on white.
    expect(contrastRatio('#767676', '#ffffff')).toBeCloseTo(4.54, 2);
  });

  it('is symmetric', () => {
    expect(contrastRatio('#2b5fb4', '#fbfaf7')).toBe(contrastRatio('#fbfaf7', '#2b5fb4'));
  });

  it('returns null when either colour is unreadable', () => {
    expect(contrastRatio('', '#fff')).toBeNull();
    expect(contrastRatio('#fff', 'transparent')).toBeNull();
  });
});

describe('meetsAA', () => {
  it('passes at the threshold and fails just below it', () => {
    expect(meetsAA(4.5, 4.5)).toBe(true);
    expect(meetsAA(4.49, 4.5)).toBe(false);
    expect(meetsAA(3, 3)).toBe(true);
  });

  it('never passes an unmeasured ratio', () => {
    expect(meetsAA(null, 3)).toBe(false);
  });
});
