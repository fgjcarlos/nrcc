/**
 * Centralized UI copy constants.
 *
 * Language policy: English only. The codebase has no i18n setup; mixing
 * languages inside one constant file (and inside a single screen) was
 * the source of the audit issue #677. Keep new entries in English.
 * If a future change introduces translation, do it via a single i18n
 * provider — not by appending Spanish strings to this object.
 */

export const UI_COPY = {
  cancel: 'Cancel',
  confirm: 'Confirm',
  confirmAction: 'Confirm Action',
  processing: 'Processing...',
  loading: 'Loading...',
  errorOccurred: 'An error occurred',
  tryAgain: 'Try again',
  delete: 'Delete',
  add: 'Add',
  saving: 'Saving...',
  saved: 'Saved',
  typeToConfirm: (word: string) => `Type "${word}" to confirm`,
  // Backups
  loadingBackups: 'Loading backups',
  readingBackups: 'Reading local snapshots and associated metadata.',
  failedToLoadBackups: 'Could not load backup list',
  retryLoadBackups: 'Retry loading or check backend status.',
  noBackupsYet: 'No backups yet',
  createFirstBackup: 'Create first backup',
  createBackupDesc: 'Create one manually or enable automatic scheduler',
  type: 'Type',
  name: 'Name',
  date: 'Date',
  files: 'Files',
  size: 'Size',
  actions: 'Actions',
  download: 'Download',
  restore: 'Restore',
  // Pagination
  previousPage: 'Previous',
  nextPage: 'Next',
  pageOf: (current: number, total: number) => `Page ${current} of ${total}`,
   // Confirmation Dialog
    confirmDeletion: 'Confirm Deletion',
    typeToConfirmDelete: (identifier: string) => `Type "${identifier}" to confirm`,
    deleteUser: 'Delete User',
    deleteUserDescription: (username: string) => `You are about to delete user "${username}". This action cannot be undone.`,
    deleteBackup: 'Delete Backup',
    deleteBackupDescription: (name: string) => `You are about to delete backup "${name}". This action cannot be undone.`,
    // User Management
    userManagement: 'User Management',
    accessDenied: 'Access Denied',
    accessDeniedDescription: "You don't have permission to view this page.",
    editUser: 'Edit User',
    changeRole: 'Change Role',
    changePassword: 'Change Password',
    createUser: 'Create User',
    roleLabel: 'Role',
    createdLabel: 'Created',
    usernameLabel: 'Username',
    passwordLabel: 'Password',
    newPasswordLabel: 'New Password',
    keepCurrentPasswordPlaceholder: 'Leave empty to keep current password',
    cannotDemoteLastAdmin: 'Cannot demote the last admin user',
    noUsersYet: 'No users yet',
    addFirstUser: 'Add the first user to get started',
    // Backup Actions
   backupConfigurationSaved: 'Backup configuration saved',
   backupCreated: 'Backup created successfully',
   backupRestored: 'Backup restored successfully',
   backupDeleted: 'Backup deleted',
   retentionPolicySaved: 'Retention policy saved',
   preRestoreBackupNotice: (id: string) => `A safety backup was created: ${id}`,
   backupIdentifierInvalid: 'The backup has no valid identifier',
   backupCronRequired: 'Enter a cron expression for the custom scheduler',
   backupDownloadStarted: 'Download started',
   backupDownloadFailed: 'Could not download the backup',
   couldNotCreateBackup: 'Could not create the backup',
   couldNotRestoreBackup: 'Could not restore the backup',
   couldNotDeleteBackup: 'Could not delete the backup',
   couldNotSaveRetentionPolicy: 'Could not save the retention policy',
   couldNotSaveBackupConfiguration: 'Could not save the backup configuration',
  // Flows
  loadingFlows: 'Loading flows',
  failedToLoadFlows: 'Failed to load flows',
  noFlowsFound: 'No flows found',
  nodeRedUnavailable: 'Node-RED is not available',
  checkNodeRedContainer: 'Verify that the Node-RED container is running.',
  selectAll: 'Select all',
  deselectAll: 'Deselect all',
  selected: (count: number) => `${count} flow${count !== 1 ? 's' : ''} selected`,
  analyzing: 'Analyzing...',
  analyzeWithAI: 'Analyze with AI',
  nodes: 'nodes',
  connections: 'connections',
  disabled: 'Disabled',
  aiAnalysis: 'AI Analysis',
  strengths: 'Strengths',
  improvements: 'Improvements',
  suggestions: 'Suggestions',
  analyzedAt: (date: string) => `Analyzed ${date}`,
  select: 'Select',
  deselect: 'Deselect',
  // Docker container actions
  restartContainerTitle: 'Restart container',
  restartContainerDesc:
    'Are you sure you want to restart the Node-RED container? The service will be briefly unavailable.',
  stopContainerTitle: 'Stop container',
  stopContainerDesc:
    'Are you sure you want to stop the Node-RED container? The service will be unavailable until you restart it.',
  // Configuration: host status block
  installationDetected: 'Installation detected',
  nodeRedNotDetected: '(no Node-RED detected)',
  pathNotDetected: 'no path detected',
  // Configuration: raw settings editor panel
  advancedSettingsTitle: 'Advanced settings.js',
  advancedSettingsDescription:
    'Edit the live settings.js file detected by nrcc. A backup is created automatically before saving.',
  lastBackup: (path: string) => `Last backup: ${path}`,
  unlockToEdit: 'Unlock to edit settings.js',
  unlockDialogTitle: 'Edit Node-RED settings.js directly',
  unlockDialogDescription: (backupPath: string) =>
    `This change is written directly to the host settings.js file (${backupPath}). A backup is taken automatically before saving. ` +
    'A syntax error in settings.js can prevent Node-RED from starting, which would take the orchestrator offline.',
  unlockDialogAcknowledgement:
    'I understand that a bad save can break Node-RED startup.',
  saveChanges: 'Save changes',
  cancelEdit: 'Cancel',
  saveRawSettings: 'Save raw settings.js',
  savingRawSettings: 'Saving settings.js...',
  lockedBadge: 'Locked — click "Unlock to edit" to make changes',
} as const;
