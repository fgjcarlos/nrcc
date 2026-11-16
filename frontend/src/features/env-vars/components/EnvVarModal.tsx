import React, { useState, useEffect, useId } from 'react';
import { type EnvVar } from '@/features/env-vars/services/envService';
import { useT } from '@/i18n';
import { Button, Dialog, DialogBody, DialogFooter, Input, Select } from '@/shared/components/ui';
import { fieldControlClass } from '@/shared/components/ui/useField';

export function EnvVarModal({
  formData,
  setFormData,
  onCancel,
  onSubmit,
  editing,
  isPending = false,
}: {
  formData: { key: string; value: string; type: EnvVar['type']; description: string };
  setFormData: (v: any) => void;
  onCancel: () => void;
  onSubmit: (e: React.FormEvent) => void;
  editing: boolean;
  isPending?: boolean;
}) {
  const { t, i18n } = useT();
  // Form-level validation state
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  // The shared Dialog locks document scroll, traps focus, closes on Escape and
  // returns focus to the invoking control. The modal is only mounted while open.
  const valueId = useId();
  const valueErrorId = `${valueId}-error`;

  // Validate form on formData changes. Depends on the active language
  // instead of the t() reference so the effect re-runs only when the
  // resolved strings would actually change, satisfying the
  // react-hooks/exhaustive-deps rule without a per-render churn.
  useEffect(() => {
    const errors: Record<string, string> = {};

    // Key is required
    if (!formData.key.trim()) {
      errors.key = t('env-vars:modalKeyRequired');
    }

    // Value validation based on type
    if (!editing && !formData.value.trim()) {
      errors.value = t('env-vars:modalValueRequired');
    }

    if (formData.value && formData.type === 'number') {
      if (Number.isNaN(Number(formData.value))) {
        errors.value = t('env-vars:modalValueMustBeNumber');
      }
    }

    if (formData.value && formData.type === 'boolean') {
      const lower = formData.value.toLowerCase();
      if (lower !== 'true' && lower !== 'false') {
        errors.value = t('env-vars:modalValueMustBeBoolean');
      }
    }

    setValidationErrors(errors);
  }, [formData, editing, t, i18n.language]);

  // Handle type change - clear incompatible values
  const handleTypeChange = (newType: EnvVar['type']) => {
    let newValue = formData.value;

    // When changing away from boolean, clear the value
    if (formData.type === 'boolean' && newType !== 'boolean') {
      newValue = '';
    }
    // When changing to boolean from another type, set default to false
    else if (formData.type !== 'boolean' && newType === 'boolean') {
      newValue = '';
    }
    // When changing to number, clear if current value is non-numeric
    else if (newType === 'number' && formData.value && Number.isNaN(Number(formData.value))) {
      newValue = '';
    }

    setFormData({ ...formData, type: newType, value: newValue });
  };

  // Render type-aware value input
  const renderValueInput = () => {
    const isRequired = !editing && !formData.value;
    // Raw inputs take the shared text-control boundary from src/index.css.
    const textProps = {
      id: valueId,
      value: formData.value,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) => setFormData({ ...formData, value: e.target.value }),
      placeholder: editing ? '(leave empty to keep current)' : '',
      className: fieldControlClass,
      required: isRequired,
      'aria-invalid': validationErrors.value ? true : undefined,
      'aria-describedby': validationErrors.value ? valueErrorId : undefined,
    };

    switch (formData.type) {
      case 'number':
        return <input type="number" {...textProps} />;

      case 'boolean':
        return (
          <div className="flex min-h-12 w-full items-center justify-between gap-3">
            <span className="text-sm text-ds-text-secondary">{t('env-vars:modalValueLabel')}:</span>
            <label className="flex items-center gap-2">
              <span className="sr-only">{t('env-vars:modalBooleanValueLabel')}</span>
              <input
                id={valueId}
                type="checkbox"
                checked={formData.value === 'true'}
                onChange={(e) => setFormData({ ...formData, value: e.target.checked ? 'true' : 'false' })}
                className="toggle toggle-primary toggle-md"
              />
              <span className="badge badge-primary text-xs font-bold">
                {formData.value === 'true' ? 'true' : 'false'}
              </span>
            </label>
          </div>
        );

      case 'secret':
        return <input type="password" autoComplete="off" {...textProps} />;

      case 'string':
      default:
        return <input type="text" {...textProps} />;
    }
  };

  // Check if form can be submitted
  const canSubmit = Object.keys(validationErrors).length === 0;

  return (
    <Dialog
      open
      onClose={onCancel}
      title={editing ? t('env-vars:modalEditTitle') : t('env-vars:modalNewTitle')}
    >
      <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
        <DialogBody className="space-y-4">
          <Input
            label={t('env-vars:modalKeyLabel')}
            type="text"
            value={formData.key}
            onChange={(e) => setFormData({ ...formData, key: e.target.value.toUpperCase() })}
            disabled={editing}
            placeholder="MY_VARIABLE"
            className="font-mono"
            error={validationErrors.key}
            required
          />

          {/* Type comes before Value: it decides the value control. */}
          <Select
            label={t('env-vars:modalTypeLabel')}
            value={formData.type}
            onChange={(e) => handleTypeChange(e.target.value as EnvVar['type'])}
          >
            <option value="string">{t('env-vars:typeString')}</option>
            <option value="number">{t('env-vars:typeNumber')}</option>
            <option value="boolean">{t('env-vars:typeBoolean')}</option>
            <option value="secret">{t('env-vars:typeSecret')}</option>
          </Select>

          {/* Type-aware Value field */}
          <div className="space-y-1">
            <label htmlFor={valueId} className="block text-xs text-ds-text-secondary">
              {t('env-vars:modalValueFieldLabel')}
            </label>
            <div data-testid="env-var-value-field" className="min-h-12 transition-all duration-150">
              {renderValueInput()}
            </div>
            {validationErrors.value && (
              <p id={valueErrorId} className="text-xs text-error">{validationErrors.value}</p>
            )}
          </div>

          <Input
            label={t('env-vars:modalDescriptionLabel')}
            type="text"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="What this variable is for"
          />
        </DialogBody>

        <DialogFooter>
          <Button type="button" variant="secondary" onClick={onCancel} disabled={isPending}>
            {t('common:cancel')}
          </Button>
          <Button type="submit" variant="solid" disabled={!canSubmit || isPending} className="relative">
            {/* The label keeps the button width while the spinner shows. */}
            <span className={isPending ? 'opacity-0' : undefined}>{t('common:save')}</span>
            {isPending && (
              <span className="absolute inset-0 flex items-center justify-center">
                <span className="loading loading-spinner loading-sm"></span>
              </span>
            )}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}

export default EnvVarModal;
