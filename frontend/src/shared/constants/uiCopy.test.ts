import { describe, it, expect } from 'vitest';
import { UI_COPY } from './uiCopy';

// Build-time guard for #482: every toast string consumed by the
// backups feature must live in UI_COPY as a non-empty string (or a
// function returning one). Missing keys would render `undefined` in
// the toast at runtime; this test catches that before merge.
describe('UI_COPY — backup copy keys', () => {
    const stringKeys = [
        'backupConfigurationSaved',
        'backupCreated',
        'backupRestored',
        'backupDeleted',
        'retentionPolicySaved',
        'backupIdentifierInvalid',
        'backupCronRequired',
        'backupDownloadStarted',
        'backupDownloadFailed',
        'couldNotCreateBackup',
        'couldNotRestoreBackup',
        'couldNotDeleteBackup',
        'couldNotSaveRetentionPolicy',
        'couldNotSaveBackupConfiguration',
    ] as const;

    it.each(stringKeys)('has non-empty string %s', (key) => {
        const value = UI_COPY[key];
        expect(typeof value).toBe('string');
        expect((value as string).length).toBeGreaterThan(0);
    });

    it('preRestoreBackupNotice is a function returning a non-empty string', () => {
        expect(typeof UI_COPY.preRestoreBackupNotice).toBe('function');
        const sample = UI_COPY.preRestoreBackupNotice('abc-123');
        expect(typeof sample).toBe('string');
        expect(sample.length).toBeGreaterThan(0);
        expect(sample).toContain('abc-123');
    });
});
