import { describe, expect, it } from 'vitest';
import en from './en.json';
import sv from './sv.json';

describe('i18n keys', () => {
  it('sv and en share required keys', () => {
    const keys = ['appTitle', 'gdpr', 'addSymbol', 'alerts', 'history'];
    for (const k of keys) {
      expect(en).toHaveProperty(k);
      expect(sv).toHaveProperty(k);
    }
  });
});
