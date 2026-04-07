import { FullAppConfig, FieldError } from '../../types/config'

const ALLOWED_LOG_LEVELS = ['fatal', 'error', 'warn', 'info', 'debug', 'trace']

export function validateFullConfig(config: FullAppConfig): FieldError[] {
  const errors: FieldError[] = []

  // Server validation
  if (config.server.uiPort < 1 || config.server.uiPort > 65535) {
    errors.push({
      field: 'server.uiPort',
      message: 'Port must be between 1 and 65535',
    })
  }

  // Security validation
  if (
    config.security.credentialSecret &&
    config.security.credentialSecret.length < 12
  ) {
    errors.push({
      field: 'security.credentialSecret',
      message: 'Credential secret must be at least 12 characters',
    })
  }

  // Flows validation
  if (!config.flows.flowFile.endsWith('.json')) {
    errors.push({
      field: 'flows.flowFile',
      message: 'Flow file must end with .json',
    })
  }

  if (config.flows.flowFile.includes('/') || config.flows.flowFile.includes('\\')) {
    errors.push({
      field: 'flows.flowFile',
      message: 'Flow file cannot contain path separators',
    })
  }

  // Logging validation
  if (!ALLOWED_LOG_LEVELS.includes(config.logging.console.level)) {
    errors.push({
      field: 'logging.console.level',
      message: `Log level must be one of: ${ALLOWED_LOG_LEVELS.join(', ')}`,
    })
  }

  // HTTPS validation
  if (config.https.enabled) {
    if (!config.https.keyFile) {
      errors.push({
        field: 'https.keyFile',
        message: 'Key file is required when HTTPS is enabled',
      })
    }
    if (!config.https.certFile) {
      errors.push({
        field: 'https.certFile',
        message: 'Certificate file is required when HTTPS is enabled',
      })
    }
  }

  return errors
}
