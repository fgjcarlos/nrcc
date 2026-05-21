/**
 * Cron expression utilities for backup scheduler
 */

// Basic 5-field cron validation
// Simplified: allows numbers, *, /, -, comma separators
// Validates field ranges: min (0-59), hr (0-23), dom (1-31), mon (1-12), dow (0-6)
export function validateCron(expr: string): boolean {
  const trimmed = expr.trim();
  
  // Must be exactly 5 fields separated by spaces
  const fields = trimmed.split(/\s+/);
  if (fields.length !== 5) {
    return false;
  }

  // Basic check: each field should only contain numbers, *, /, -, or comma
  const validFieldRegex = /^[\d,*/-]+$/;
  if (!fields.every(field => validFieldRegex.test(field))) {
    return false;
  }

  // More strict: validate ranges for each field
  // minute: 0-59, hour: 0-23, day: 1-31, month: 1-12, dow: 0-6
  const [minute, hour, day, month, dow] = fields;
  const ranges = [
    { field: minute, min: 0, max: 59 },
    { field: hour, min: 0, max: 23 },
    { field: day, min: 1, max: 31 },
    { field: month, min: 1, max: 12 },
    { field: dow, min: 0, max: 6 },
  ];

  for (const { field, min, max } of ranges) {
    if (field === '*') continue;

    // Extract all numbers from the field
    const numbers = field.split(/[,/-]/).filter(n => n.length > 0 && n !== '*');
    
    for (const numStr of numbers) {
      const num = Number(numStr);
      if (!Number.isInteger(num) || num < min || num > max) {
        return false;
      }
    }
  }

  return true;
}
