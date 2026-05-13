import React, { useState, useEffect } from 'react';
import { type EnvVar } from '@/features/env-vars/services/envService';

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
  // Form-level validation state
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  // Validate form on formData changes
  useEffect(() => {
    const errors: Record<string, string> = {};

    // Key is required
    if (!formData.key.trim()) {
      errors.key = 'Key is required';
    }

    // Value validation based on type
    if (!editing && !formData.value.trim()) {
      errors.value = 'Value is required';
    }

    if (formData.value && formData.type === 'number') {
      if (Number.isNaN(Number(formData.value))) {
        errors.value = 'Value must be a valid number';
      }
    }

    if (formData.value && formData.type === 'boolean') {
      const lower = formData.value.toLowerCase();
      if (lower !== 'true' && lower !== 'false') {
        errors.value = 'Value must be true or false';
      }
    }

    setValidationErrors(errors);
  }, [formData, editing]);

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

    switch (formData.type) {
      case 'number':
        return (
          <input
            type="number"
            value={formData.value}
            onChange={(e) => setFormData({ ...formData, value: e.target.value })}
            placeholder={editing ? '(leave empty to keep current)' : ''}
            className="glass-panel w-full h-full rounded-xl border-0 px-3 py-2 text-base-content focus:outline-none"
            required={isRequired}
          />
        );

      case 'boolean':
        return (
          <div className="flex w-full items-center justify-between gap-3 px-3 py-3">
            <span className="text-sm font-semibold text-base-content/70 uppercase tracking-wider">Value:</span>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.value === 'true'}
                onChange={(e) => setFormData({ ...formData, value: e.target.checked ? 'true' : 'false' })}
                className="toggle toggle-primary toggle-lg"
              />
              <span className={`text-sm font-bold ${formData.value === 'true' ? 'text-primary' : 'text-base-content/50'}`}>
                {formData.value === 'true' ? 'true' : 'false'}
              </span>
            </div>
          </div>
        );

      case 'secret':
        return (
          <input
            type="password"
            value={formData.value}
            onChange={(e) => setFormData({ ...formData, value: e.target.value })}
            placeholder={editing ? '(leave empty to keep current)' : ''}
            className="glass-panel w-full h-full rounded-xl border-0 px-3 py-2 text-base-content focus:outline-none"
            required={isRequired}
          />
        );

      case 'string':
      default:
        return (
          <input
            type="text"
            value={formData.value}
            onChange={(e) => setFormData({ ...formData, value: e.target.value })}
            placeholder={editing ? '(leave empty to keep current)' : ''}
            className="glass-panel w-full h-full rounded-xl border-0 px-3 py-2 text-base-content focus:outline-none"
            required={isRequired}
          />
        );
    }
  };

  // Check if form can be submitted
  const canSubmit = Object.keys(validationErrors).length === 0;

  return (
    <div className="modal-overlay">
      <div className="surface-panel w-full max-w-md border border-border p-6 shadow-glow">
        <div className="mb-4 flex items-start justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Environment</p>
            <h2 className="text-xl font-bold text-base-content">{editing ? 'Edit Variable' : 'New Variable'}</h2>
          </div>
        </div>
        <form onSubmit={onSubmit} className="space-y-5">
          {/* Key field */}
          <div>
            <label className="mb-2 block text-xs font-semibold text-base-content/70 uppercase tracking-wider">Key</label>
            <input
              type="text"
              value={formData.key}
              onChange={(e) => setFormData({ ...formData, key: e.target.value.toUpperCase() })}
              disabled={editing}
              placeholder="MY_VARIABLE"
              className={`glass-panel w-full rounded-xl border px-3 py-2 text-base-content disabled:opacity-50 ${
                validationErrors.key ? 'border-error' : 'border-border'
              }`}
              required
            />
            {validationErrors.key && (
              <p className="mt-1 text-xs text-error">{validationErrors.key}</p>
            )}
          </div>

          {/* Type selector (moved before Value) */}
          <div>
            <label className="mb-2 block text-xs font-semibold text-base-content/70 uppercase tracking-wider">Type</label>
            <select
              value={formData.type}
              onChange={(e) => handleTypeChange(e.target.value as EnvVar['type'])}
              className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content"
            >
              <option value="string">String</option>
              <option value="number">Number</option>
              <option value="boolean">Boolean</option>
              <option value="secret">Secret (encrypted)</option>
            </select>
          </div>

          {/* Type-aware Value field */}
          <div>
            <label className="mb-2 block text-xs font-semibold text-base-content/70 uppercase tracking-wider">Value</label>
            <div
              className={`rounded-xl border min-h-[44px] flex items-center ${
                validationErrors.value ? 'border-error' : 'border-border'
              }`}
            >
              {renderValueInput()}
            </div>
            {validationErrors.value && (
              <p className="mt-1 text-xs text-error">{validationErrors.value}</p>
            )}
          </div>

          {/* Description field */}
          <div>
            <label className="mb-2 block text-xs font-semibold text-base-content/70 uppercase tracking-wider">Description (optional)</label>
            <input
              type="text"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              placeholder="What this variable is for"
              className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content"
            />
          </div>

          {/* Action buttons */}
          <div className="flex justify-end gap-2 pt-4">
            <button
              type="button"
              onClick={onCancel}
              disabled={isPending}
              className="action-btn-secondary disabled:opacity-50"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={!canSubmit || isPending}
              className="action-btn-primary relative disabled:opacity-50"
            >
              {isPending ? (
                <>
                  <span className="opacity-0">Guardar</span>
                  <span className="absolute inset-0 flex items-center justify-center">
                    <span className="loading loading-spinner loading-sm"></span>
                  </span>
                </>
              ) : (
                'Guardar'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default EnvVarModal;
