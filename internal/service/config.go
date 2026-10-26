package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

var ErrAdminAuthPlaintextPassword = errors.New("adminAuth password must be a bcrypt hash")

// ConfigService handles Node-RED configuration
type ConfigService struct {
	store   *store.JSONStore[model.NodeRedConfig]
	dataDir string
	hostSvc *HostService
}

// NewConfigService creates a new config service
func NewConfigService(dataDir string) *ConfigService {
	return NewConfigServiceWithHost(dataDir, NewHostService(dataDir))
}

// NewIsolatedConfigService creates a config service whose settings.js writes are
// confined to dataDir. It is intended for hermetic tests.
func NewIsolatedConfigService(dataDir string) *ConfigService {
	return NewConfigServiceWithHost(dataDir, NewIsolatedHostService(dataDir))
}

// NewConfigServiceWithHost creates a config service with shared host detection.
func NewConfigServiceWithHost(dataDir string, hostSvc *HostService) *ConfigService {
	configPath := filepath.Join(dataDir, "config.json")
	jsonStore := store.NewJSONStore[model.NodeRedConfig](configPath)

	return &ConfigService{
		store:   jsonStore,
		dataDir: dataDir,
		hostSvc: hostSvc,
	}
}

// Get retrieves the current configuration
func (s *ConfigService) Get() (model.NodeRedConfig, error) {
	if s.store.Exists() {
		cfg, err := s.store.Read()
		if err != nil {
			return cfg, err
		}
		s.decorateConfig(&cfg)
		return cfg, nil
	}

	if cfg, ok, err := s.readFromSettingsFile(); err != nil {
		return cfg, err
	} else if ok {
		s.decorateConfig(&cfg)
		return cfg, nil
	}

	// Return defaults if no config exists
	cfg := s.GetDefault()
	s.decorateConfig(&cfg)
	return cfg, nil
}

// Update atomically mutates the persisted Node-RED configuration and returns
// the snapshot whose JSON commit completed successfully.
func (s *ConfigService) Update(fn func(*model.NodeRedConfig) error) (model.NodeRedConfig, error) {
	if fn == nil {
		return model.NodeRedConfig{}, fmt.Errorf("config update callback is nil")
	}
	var committed model.NodeRedConfig
	err := s.store.Update(func(current *model.NodeRedConfig) error {
		if err := fn(current); err != nil {
			return err
		}
		committed = *current
		return nil
	})
	if err != nil {
		return model.NodeRedConfig{}, err
	}
	return committed, nil
}

// Save atomically commits the configuration before rendering settings.js.
// If settings.js rendering or writing fails, the returned error explicitly
// reports the post-commit failure; the durable JSON configuration is not rolled back.
//
// EnvVars ownership: when the caller passes EnvVars in cfg, those are honored as
// authoritative for the committed slice. EnvService.Set/Delete are the supported
// path for env-var mutations and also route through this same Update, so concurrent
// Save+Set on the same store coordinate via the JSONStore lock.
// upgradePlaintextAdminAuthPasswords hashes any cleartext password sent by
// the frontend. The UI never hashes (and shouldn't — exposing a hashing
// primitive to the browser would weaken the contract), so the only safe
// interpretation of a non-bcrypt password on the wire is "this is the
// new cleartext password the admin wants to set". Hashing here also makes
// the autofill scenario (#706) harmless: even if a browser autofills the
// real password after the admin types a different one, that cleartext is
// exactly what would be stored anyway, so the round-trip is identity.
func upgradePlaintextAdminAuthPasswords(cfg model.NodeRedConfig) error {
	if cfg.AdminAuth == nil {
		return nil
	}
	for i, user := range cfg.AdminAuth.Users {
		if user.Password == "" || isBcryptHash(user.Password) {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), BcryptCost)
		if err != nil {
			return fmt.Errorf("adminAuth user %d (%q): %w", i, user.Username, err)
		}
		cfg.AdminAuth.Users[i].Password = string(hash)
	}
	return nil
}

// Save commits the configuration to the JSON store and renders settings.js.
// Plaintext passwords in AdminAuth are upgraded to bcrypt hashes here so the
// frontend can submit cleartext when changing the admin password without the
// validation rejecting it (#706).
func (s *ConfigService) Save(cfg model.NodeRedConfig) error {
	fallbacks := s.securityFallbacks(&cfg)
	committed, err := s.Update(func(current *model.NodeRedConfig) error {
		candidate := cfg
		preserveAdminAuthPasswords(current, &candidate)
		// Issue #881: a payload that omits a security surface keeps the
		// loaded one instead of clearing it.
		preserveSecuritySurfaces(current, fallbacks, &candidate)
		preserveWriteOnlySecrets(current, &candidate)
		if err := upgradePlaintextAdminAuthPasswords(candidate); err != nil {
			return err
		}
		if err := validateAdminAuthPasswords(candidate); err != nil {
			return err
		}
		if err := s.Validate(candidate); err != nil {
			return err
		}
		s.decorateConfig(&candidate)
		*current = candidate
		return nil
	})
	if err != nil {
		return err
	}

	if _, err := s.writeSettingsFile(s.renderSettings(committed), false); err != nil {
		return fmt.Errorf("config JSON committed but settings.js update failed: %w", err)
	}
	return nil
}

// StructuredApplyInput captures the config state committed as one step of
// a coordinator-owned structured apply. Previous and Committed support
// field-aware recovery without overwriting unrelated concurrent updates.
type StructuredApplyInput struct {
	// Previous is the authoritative config snapshot captured under the
	// JSONStore update lock before this apply's fields were published.
	Previous model.NodeRedConfig
	// Committed is the post-store NodeRedConfig the operator
	// should see in the response (passwords are bcrypt hashes
	// at this point).
	Committed model.NodeRedConfig
	// Content is the rendered settings.js bytes the
	// coordinator's apply stage will atomically write.
	Content string
	// Live is the live SettingsDocument at the moment of the
	// commit. Path is the on-disk settings.js location;
	// Revision is the fingerprint the coordinator's revision
	// precondition compares against req.Expected.
	Live model.SettingsDocument
	// BackupDir is the directory the apply stage writes the
	// pre-write snapshot to. Matches the directory the legacy
	// Save() flow used so existing backup tooling keeps finding
	// the new files.
	BackupDir string
}

// CommitStructured validates and atomically publishes the structured config
// while capturing the prior snapshot under JSONStore's update lock. Callers
// must invoke it only after acquiring the apply coordinator's single-flight
// guard and passing the settings revision precondition. The coordinator owns
// restoring Previous if any later apply or runtime stage fails.
//
// The full-replacement contract is kept for the configuration editor
// (POST /api/config). The apply endpoint uses CommitStructuredPatch so a
// patch that carries a single surface does not drop the other settings.
func (s *ConfigService) CommitStructured(cfg model.NodeRedConfig) (StructuredApplyInput, error) {
	return s.commitStructured(&cfg, nil)
}

// CommitStructuredPatch merges a presence-aware patch onto the live
// configuration and publishes the result while capturing the prior snapshot
// under JSONStore's update lock. Callers must invoke it only after acquiring
// the apply coordinator's single-flight guard and passing the settings
// revision precondition.
//
// The merge reads the live configuration inside the store update lock, so
// the patch is applied to the state the store holds at commit time and a
// concurrent environment-variable write cannot slip between the read and
// the write. The coordinator owns restoring Previous if any later apply or
// runtime stage fails.
func (s *ConfigService) CommitStructuredPatch(patch model.ConfigPatch) (StructuredApplyInput, error) {
	return s.commitStructured(nil, &patch)
}

// PreviewStructuredPatch computes the merged configuration without
// committing it. The runtime adapter needs the candidate port before the
// transaction starts, and the port only exists once the patch has been
// merged onto the live configuration.
//
// The preview reads the live configuration outside the commit lock; the
// commit recomputes the merge under the lock. Both merges read the same
// live state, and the store has a single writer per deployment, so the
// previewed port and the committed port agree.
func (s *ConfigService) PreviewStructuredPatch(patch model.ConfigPatch) (model.NodeRedConfig, error) {
	base, _, err := s.patchBase()
	if err != nil {
		return model.NodeRedConfig{}, err
	}
	return MergePatch(base, patch)
}

// commitStructured publishes either a full configuration (full != nil) or a
// patch merged onto the live configuration (patch != nil).
func (s *ConfigService) commitStructured(full *model.NodeRedConfig, patch *model.ConfigPatch) (StructuredApplyInput, error) {
	live, err := s.GetRawSettings()
	if err != nil {
		return StructuredApplyInput{}, err
	}
	// The merge base is read before the update lock is taken: the JSON
	// store forbids re-entering the store from inside an update callback.
	base, hasBase, err := s.patchBase()
	if err != nil {
		return StructuredApplyInput{}, err
	}
	// Issue #881: the full-replacement contract keeps a security surface the
	// payload omits. The fallback lookup happens outside the update lock
	// because it reads the live settings.js.
	fallbacks := s.securityFallbacks(full)
	var previous model.NodeRedConfig
	committed, err := s.Update(func(current *model.NodeRedConfig) error {
		previous = *current
		candidate := model.NodeRedConfig{}
		// The preservation source is the live configuration the candidate
		// was derived from: the merge base for a patch, the committed
		// state for a full replacement.
		existing := current
		if patch != nil {
			mergeBase := *current
			if !hasBase {
				mergeBase = base
			}
			merged, err := MergePatch(mergeBase, *patch)
			if err != nil {
				return err
			}
			candidate = merged
			existing = &base
		} else {
			candidate = *full
			// Issue #881: a payload that omits a security surface keeps the
			// loaded one instead of clearing it.
			preserveSecuritySurfaces(current, fallbacks, &candidate)
		}
		preserveAdminAuthPasswords(existing, &candidate)
		preserveWriteOnlySecrets(existing, &candidate)
		if err := upgradePlaintextAdminAuthPasswords(candidate); err != nil {
			return err
		}
		if err := validateAdminAuthPasswords(candidate); err != nil {
			return err
		}
		if err := s.Validate(candidate); err != nil {
			return err
		}
		// Keep state owned by EnvService and other configuration readers out
		// of the structured apply's write set.
		candidate.EnvVars = current.EnvVars
		candidate.FunctionLibraries = current.FunctionLibraries
		candidate.RuntimeState = current.RuntimeState
		candidate.SettingsPath = current.SettingsPath
		candidate.SettingsSource = current.SettingsSource
		s.decorateConfig(&candidate)
		*current = candidate
		return nil
	})
	if err != nil {
		return StructuredApplyInput{}, err
	}
	return StructuredApplyInput{
		Previous:  previous,
		Committed: committed,
		Content:   s.renderSettings(committed),
		Live:      live,
		BackupDir: filepath.Join(s.dataDir, "backups", "settings"),
	}, nil
}

// patchBase returns the live configuration a patch merges onto, plus a flag
// that reports whether the persisted store supplied it.
//
// The persisted store wins when it exists: it is NRCC's own authoritative
// configuration. A deployment whose store does not exist yet falls back to
// the configuration parsed from the live settings.js, so the first patch
// keeps the port and the roots the running Node-RED already uses. The
// defaults are the last resort.
//
// The store lookup happens outside the update lock because the JSON store
// forbids re-entering the store from inside an update callback.
func (s *ConfigService) patchBase() (model.NodeRedConfig, bool, error) {
	if s.store.Exists() {
		cfg, err := s.store.Read()
		if err != nil {
			return model.NodeRedConfig{}, false, fmt.Errorf("read persisted configuration: %w", err)
		}
		return cfg, true, nil
	}
	if cfg, ok, err := s.readFromSettingsFile(); err != nil {
		return model.NodeRedConfig{}, false, err
	} else if ok {
		return cfg, false, nil
	}
	return s.GetDefault(), false, nil
}

// RestoreStructured restores only fields still carrying this transaction's
// committed values. Concurrent unrelated writes therefore survive recovery.
func (s *ConfigService) RestoreStructured(state StructuredApplyInput) error {
	return s.store.Update(func(current *model.NodeRedConfig) error {
		restoreUnchangedFields(current, state.Previous, state.Committed, structuredRestoreSkip)
		return nil
	})
}

// structuredRestoreSkip lists the fields the structured apply does not own:
// the state of the other services and the internal bookkeeping.
var structuredRestoreSkip = map[string]struct{}{
	"EnvVars": {}, "FunctionLibraries": {}, "RuntimeState": {},
	"SettingsPath": {}, "SettingsSource": {}, "Extra": {},
}

// mirrorRestoreSkip lists the fields the mirror reconciliation does not own.
var mirrorRestoreSkip = map[string]struct{}{
	"FunctionLibraries": {}, "RuntimeState": {}, "SettingsPath": {},
	"SettingsSource": {}, "Extra": {},
}

// restoreUnchangedFields copies the previous value of every exported field
// that still holds the committed value.
//
// Unexported fields are skipped: they are internal bookkeeping (the
// security-surface presence of issue #881) and reflect.Value.Interface
// panics when it is called on them.
func restoreUnchangedFields(current *model.NodeRedConfig, previous, committed model.NodeRedConfig, skip map[string]struct{}) {
	live := reflect.ValueOf(current).Elem()
	livePrevious := reflect.ValueOf(previous)
	liveCommitted := reflect.ValueOf(committed)
	for i := 0; i < live.NumField(); i++ {
		field := live.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		if _, ok := skip[field.Name]; ok {
			continue
		}
		if reflect.DeepEqual(live.Field(i).Interface(), liveCommitted.Field(i).Interface()) {
			live.Field(i).Set(livePrevious.Field(i))
		}
	}
}

// preserveWriteOnlySecrets keeps the stored value of the write-only
// configuration secrets when the candidate does not carry one.
//
// The configuration API never returns credentialSecret or the TLS
// passphrase (issue #877), so a client that reads the configuration and
// writes it back submits an empty (or redacted) value for those fields.
// Treating that as "clear the secret" would silently destroy the stored
// credential-encryption secret and the TLS passphrase on every save.
//
// A candidate that carries a value rotates the secret, and the literal
// "false" still disables credential encryption: only empty or redacted
// placeholders mean "keep what is stored".
//
// The HTTPS block is cloned before the passphrase is restored so the
// caller's configuration value is never mutated through the shared
// pointer.
func preserveWriteOnlySecrets(existing, candidate *model.NodeRedConfig) {
	if candidate == nil || existing == nil {
		return
	}
	if model.IsRedactedSecret(candidate.CredentialSecret) {
		candidate.CredentialSecret = existing.CredentialSecret
	}
	if candidate.Https != nil && existing.Https != nil &&
		model.IsRedactedSecret(candidate.Https.Passphrase) {
		https := *candidate.Https
		https.Passphrase = existing.Https.Passphrase
		candidate.Https = &https
	}
}

func preserveAdminAuthPasswords(existing, candidate *model.NodeRedConfig) {
	if candidate.AdminAuth == nil || len(candidate.AdminAuth.Users) == 0 ||
		existing.AdminAuth == nil || len(existing.AdminAuth.Users) == 0 {
		return
	}
	for i, newUser := range candidate.AdminAuth.Users {
		if newUser.Password != "" {
			continue
		}
		for _, existingUser := range existing.AdminAuth.Users {
			if existingUser.Username == newUser.Username {
				candidate.AdminAuth.Users[i].Password = existingUser.Password
				break
			}
		}
	}
}

// securitySurfaceFallbacks holds the authentication surfaces loaded from the
// live settings.js. They are the fallback source when nrcc's own store does
// not know a surface yet, for example when the operator wrote adminAuth by
// hand before the first save.
type securitySurfaceFallbacks struct {
	adminAuth      *model.AdminAuth
	httpNodeAuth   *model.HTTPBasicAuth
	httpStaticAuth *model.HTTPBasicAuth
}

// preserveSecuritySurfaces keeps the authentication surfaces a payload did
// not carry, and the password hashes of the accounts it did.
//
// Issue #881: the configuration form submits only the capabilities it
// edits. A payload that omits `adminAuth` (or a basic-auth surface) must
// leave the loaded surface untouched. Before this rule existed, the
// configuration form rebuilt `adminAuth` from a single account with
// wildcard permissions, so an unrelated visual edit replaced Node-RED's
// complete authentication model: every other account, every `read`
// permission and the session expiry were dropped.
//
// Sources, in order:
//  1. the persisted nrcc store — what the operator last saved,
//  2. the live settings.js — what Node-RED is actually running with.
func preserveSecuritySurfaces(current *model.NodeRedConfig, fallbacks *securitySurfaceFallbacks, candidate *model.NodeRedConfig) {
	if candidate.AdminAuthCarried() {
		// The payload carries the surface: keep the password hash of every
		// account the operator did not change. An empty password means
		// "unchanged", exactly like the security cards send it.
		preserveAdminAuthPasswords(current, candidate)
		fillAdminAuthPasswords(fallbacks.adminAuth, candidate.AdminAuth)
	} else {
		// The payload omits the surface: keep the loaded one.
		candidate.AdminAuth = current.AdminAuth
		if candidate.AdminAuth == nil {
			candidate.AdminAuth = fallbacks.adminAuth
		}
	}

	preserveBasicAuthSurface(current.HTTPNodeAuth, fallbacks.httpNodeAuth, candidate.HTTPNodeAuthCarried(), &candidate.HTTPNodeAuth)
	preserveBasicAuthSurface(current.HTTPStaticAuth, fallbacks.httpStaticAuth, candidate.HTTPStaticAuthCarried(), &candidate.HTTPStaticAuth)
}

// fillAdminAuthPasswords copies the password hash of every candidate
// account without one from the first source that knows the username.
func fillAdminAuthPasswords(source, candidate *model.AdminAuth) {
	if source == nil || candidate == nil {
		return
	}
	for i := range candidate.Users {
		if candidate.Users[i].Password != "" {
			continue
		}
		for _, existing := range source.Users {
			if existing.Username == candidate.Users[i].Username && existing.Password != "" {
				candidate.Users[i].Password = existing.Password
				break
			}
		}
	}
}

// preserveBasicAuthSurface keeps a basic-auth surface a payload omitted and
// the bcrypt hash of a pass the operator did not change.
func preserveBasicAuthSurface(current, fallback *model.HTTPBasicAuth, carried bool, candidate **model.HTTPBasicAuth) {
	if !carried {
		if current != nil {
			*candidate = current
		} else if fallback != nil {
			*candidate = fallback
		}
		return
	}
	if *candidate == nil || (*candidate).Pass != "" {
		return
	}
	for _, source := range []*model.HTTPBasicAuth{current, fallback} {
		if source != nil && source.User == (*candidate).User && source.Pass != "" {
			(*candidate).Pass = source.Pass
			break
		}
	}
}

// securityFallbacks returns the live settings.js authentication surfaces
// needed to preserve a surface the payload omitted.
//
// The live settings.js is only parsed when the persisted store cannot
// supply the surface: the parse runs the settings sandbox and is far more
// expensive than reading the store.
func (s *ConfigService) securityFallbacks(cfg *model.NodeRedConfig) *securitySurfaceFallbacks {
	fallbacks := &securitySurfaceFallbacks{}
	if cfg == nil {
		// The patch path merges onto the live configuration, so the merge
		// itself resolves the presence of every surface.
		return fallbacks
	}
	if cfg.AdminAuthCarried() && cfg.HTTPNodeAuthCarried() && cfg.HTTPStaticAuthCarried() {
		// Every surface was carried by the payload: nothing to preserve.
		return fallbacks
	}

	persisted, err := s.store.Read()
	if err != nil {
		persisted = model.NodeRedConfig{}
	}
	needAdmin := !cfg.AdminAuthCarried() && persisted.AdminAuth == nil
	needNode := !cfg.HTTPNodeAuthCarried() && persisted.HTTPNodeAuth == nil
	needStatic := !cfg.HTTPStaticAuthCarried() && persisted.HTTPStaticAuth == nil
	if !needAdmin && !needNode && !needStatic {
		return fallbacks
	}

	doc, err := s.GetRawSettings()
	if err != nil || doc.Content == "" {
		return fallbacks
	}
	parsed, err := s.parseConfigFromContent(doc.Content)
	if err != nil {
		return fallbacks
	}
	if needAdmin {
		fallbacks.adminAuth = parsed.AdminAuth
	}
	if needNode {
		fallbacks.httpNodeAuth = parsed.HTTPNodeAuth
	}
	if needStatic {
		fallbacks.httpStaticAuth = parsed.HTTPStaticAuth
	}
	return fallbacks
}

func validateAdminAuthPasswords(cfg model.NodeRedConfig) error {
	if cfg.AdminAuth == nil {
		return nil
	}
	for i, user := range cfg.AdminAuth.Users {
		if user.Password != "" && !isBcryptHash(user.Password) {
			return fmt.Errorf("adminAuth user %d (%q): %w", i, user.Username, ErrAdminAuthPlaintextPassword)
		}
	}
	return nil
}

func isBcryptHash(s string) bool {
	if len(s) <= 4 {
		return false
	}
	return strings.HasPrefix(s, "$2a$") ||
		strings.HasPrefix(s, "$2b$") ||
		strings.HasPrefix(s, "$2y$")
}

// preserveAdminAuthPasswords preserves existing password hashes when empty password is provided
// This allows frontend to send empty password (meaning "don't change") and backend will use the existing hash.
// It reads ONLY from nrcc's own JSON store — NOT from the live settings.js — to avoid reading from the
// system-wide ~/.node-red/settings.js, which would pollute tests and cause incorrect cross-user behaviour.
func (s *ConfigService) preserveAdminAuthPasswords(cfg *model.NodeRedConfig) error {
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) == 0 {
		return nil
	}

	// Only preserve from nrcc's own persisted store, not from the system settings.js.
	if !s.store.Exists() {
		return nil
	}
	existing, err := s.store.Read()
	if err != nil {
		return fmt.Errorf("read existing persisted settings: %w", err)
	}

	if existing.AdminAuth == nil || len(existing.AdminAuth.Users) == 0 {
		// No existing adminAuth to preserve from
		return nil
	}

	// For each user in the new config, if password is empty, use the existing password hash
	for i, newUser := range cfg.AdminAuth.Users {
		if newUser.Password == "" {
			// Find matching user in existing config by username
			for _, existingUser := range existing.AdminAuth.Users {
				if existingUser.Username == newUser.Username {
					cfg.AdminAuth.Users[i].Password = existingUser.Password
					break
				}
			}
		}
	}

	return nil
}

// GetDefault returns the default configuration
func (s *ConfigService) GetDefault() model.NodeRedConfig {
	return model.DefaultNodeRedConfig()
}

// ConfigurationCapabilities returns the active Node-RED adapter contract.
// Callers must use it before attempting a settings.js mutation.
func (s *ConfigService) ConfigurationCapabilities() model.ConfigurationCapabilities {
	if s.hostSvc == nil {
		return ResolveConfigurationCapabilities("", model.SettingsDocument{})
	}
	return s.hostSvc.Detect().Configuration
}

// GetRawSettings loads the active settings.js document.
func (s *ConfigService) GetRawSettings() (model.SettingsDocument, error) {
	status := s.hostSvc.Detect()
	doc := status.Settings
	content, err := os.ReadFile(doc.Path)
	if err != nil {
		if os.IsNotExist(err) {
			doc.Content = renderSettingsJS(s.GetDefault())
		} else {
			return doc, err
		}
	} else {
		doc.Content = string(content)
	}
	doc.Revision = FingerprintSource(doc.Content)
	return doc, nil
}

// SaveRawSettings writes the active settings.js document after backing up the previous version.
func (s *ConfigService) SaveRawSettings(content string) (model.SettingsDocument, error) {
	return s.SaveRawSettingsWithRevision(content, model.SourceRevision{})
}

// SaveRawSettingsWithRevision writes the active settings.js document only
// when the on-disk fingerprint matches expected. An empty
// expected.Fingerprint is treated as "no expectation" and the call falls
// through to the legacy SaveRawSettings behaviour, preserving backward
// compatibility for callers that have not yet been wired to capture the
// revision.
//
// When the on-disk fingerprint does not match a non-empty expected
// revision, the call returns ErrSourceRevisionMismatch without touching
// disk or creating a backup. The handler maps this to HTTP 409 Conflict
// so the operator is forced to re-read, re-render and retry instead of
// silently overwriting an external change to settings.js.
//
// Slice deferral: the renderSettings routing change that would have this
// path receive an already-patched SourcePatch result instead of a raw
// content blob is deferred to a follow-up PR. The slice-B contract is
// "bytes in, bytes out, refuse on revision mismatch" — the structured
// edit path wires in afterwards.
func (s *ConfigService) SaveRawSettingsWithRevision(content string, expected model.SourceRevision) (model.SettingsDocument, error) {
	current, err := s.currentOnDiskRevision()
	if err != nil {
		return model.SettingsDocument{}, err
	}
	if !RevisionMatches(expected, current) {
		return model.SettingsDocument{}, ErrSourceRevisionMismatch
	}

	parsed, err := s.parseConfigFromContent(content)
	if err != nil {
		// Propagate sandbox timeouts so the operator gets SETTINGS_TIMEOUT
		// rather than the file being written and breaking the read path.
		return model.SettingsDocument{}, err
	}
	if err := validateAdminAuthPasswords(parsed); err != nil {
		return model.SettingsDocument{}, err
	}
	doc, err := s.writeSettingsFile(content, true)
	if err != nil {
		return doc, err
	}
	// Re-stamp the revision against the bytes that were just persisted so
	// the caller can use the returned document as the next expected
	// revision without re-reading the file. Any subsequent external edit
	// will then trigger the conflict response on the next save.
	doc.Revision = FingerprintSource(doc.Content)
	return doc, nil
}

// currentOnDiskRevision returns the SourceRevision of the bytes currently
// on disk for the active settings.js path. When the file does not exist
// (first-write state) the revision is computed against the would-be
// default content so the caller can still compare revisions on a virgin
// install where the first save is also the first read.
func (s *ConfigService) currentOnDiskRevision() (model.SourceRevision, error) {
	doc, err := s.GetRawSettings()
	if err != nil {
		return model.SourceRevision{}, err
	}
	return doc.Revision, nil
}

func (s *ConfigService) writeSettingsFile(content string, syncStore bool) (model.SettingsDocument, error) {
	status := s.hostSvc.Detect()
	doc := status.Settings
	if doc.Path == "" {
		doc.Path = filepath.Join(s.dataDir, "settings.js")
	}
	if err := os.MkdirAll(filepath.Dir(doc.Path), 0750); err != nil {
		return doc, err
	}
	previous, readErr := os.ReadFile(doc.Path)
	backupPath, err := s.backupSettingsFile(doc.Path)
	if err != nil {
		return doc, err
	}
	if err := os.WriteFile(doc.Path, []byte(content), 0600); err != nil {
		return doc, err
	}
	if err := os.Chmod(doc.Path, 0600); err != nil {
		return doc, err
	}
	if syncStore {
		// The source is authoritative (issue #880): the structured mirror
		// is re-derived from the applied bytes. Writing the parsed source
		// wholesale would drop the NRCC-owned state the mirror also holds
		// (the environment metadata, the function libraries, the runtime
		// state).
		if _, err := s.ReconcileRawSettings(content); err != nil {
			// The transaction is rejected as a whole: the pre-write bytes
			// are restored so neither representation keeps a partial
			// effect.
			if restoreErr := restoreSourceBytes(context.Background(), doc.Path, string(previous), readErr == nil); restoreErr != nil {
				return doc, errors.Join(err, fmt.Errorf("restore source after failed reconciliation: %w", restoreErr))
			}
			return doc, err
		}
	}
	doc.BackupPath = backupPath
	doc.Content = content
	doc.Writable = true
	return doc, nil
}

// Validate validates a configuration
func (s *ConfigService) Validate(cfg model.NodeRedConfig) error {
	// Use Port if set, otherwise use UIPort
	port := cfg.Port
	if port == 0 && cfg.UIPort > 0 {
		port = cfg.UIPort
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if cfg.HTTPAdminRoot == "" {
		return fmt.Errorf("httpAdminRoot cannot be empty")
	}

	if cfg.HTTPNodeRoot == "" {
		return fmt.Errorf("httpNodeRoot cannot be empty")
	}

	// Validate Admin Auth if configured
	if cfg.AdminAuth != nil {
		if err := validateAdminAuth(cfg.AdminAuth); err != nil {
			return err
		}
	}
	if err := validateHTTPBasicAuth("httpNodeAuth", cfg.HTTPNodeAuth); err != nil {
		return err
	}
	if err := validateHTTPBasicAuth("httpStaticAuth", cfg.HTTPStaticAuth); err != nil {
		return err
	}

	return nil
}

// validateAdminAuth validates AdminAuth configuration
func validateAdminAuth(auth *model.AdminAuth) error {
	if auth.Type == "" {
		return fmt.Errorf("adminAuth type must not be empty")
	}

	if len(auth.Users) == 0 {
		return fmt.Errorf("adminAuth must have at least one user")
	}

	for i, user := range auth.Users {
		if user.Username == "" {
			return fmt.Errorf("adminAuth user %d: username must not be empty", i)
		}

		if user.Password == "" {
			return fmt.Errorf("adminAuth user %d: password must not be empty", i)
		}

		if len(user.Username) < 3 {
			return fmt.Errorf("adminAuth user %d: username must be at least 3 characters", i)
		}

		if len(user.Password) < 6 {
			return fmt.Errorf("adminAuth user %d: password must be at least 6 characters", i)
		}
		if user.Permissions != "*" && user.Permissions != "read" {
			return fmt.Errorf("adminAuth user %d: permissions must be \"*\" or \"read\"", i)
		}
	}
	if auth.SessionExpiryTime < 0 {
		return fmt.Errorf("adminAuth sessionExpiryTime must not be negative")
	}

	return nil
}

func validateHTTPBasicAuth(name string, auth *model.HTTPBasicAuth) error {
	if auth == nil {
		return nil
	}
	if auth.User == "" {
		return fmt.Errorf("%s user must not be empty", name)
	}
	if !isBcryptHash(auth.Pass) {
		return fmt.Errorf("%s pass must be a bcrypt hash", name)
	}
	return nil
}

func (s *ConfigService) decorateConfig(cfg *model.NodeRedConfig) {
	if s.hostSvc == nil {
		return
	}
	settings := s.hostSvc.Detect().Settings
	cfg.SettingsPath = settings.Path
	cfg.SettingsSource = settings.Source
}

// readFromSettingsFile tries to parse the persisted settings.js into a
// NodeRedConfig. It returns:
//
//   - (cfg, true, nil)        when the file exists and parses cleanly;
//   - (zero, false, nil)      when the file is absent, unreadable, or empty —
//     the caller falls back to defaults;
//   - (zero, false, err)      when the content is present but the sandbox
//     raised a typed error (notably ErrSandboxTimeout from #665) — the caller
//     surfaces it as SETTINGS_TIMEOUT instead of silently booting a broken
//     config.
//
// Swallowing the GetRawSettings error in the "absent/unreadable" branch is
// intentional: a missing settings.js is the default state on a fresh
// deployment, not a failure.
func (s *ConfigService) readFromSettingsFile() (model.NodeRedConfig, bool, error) {
	doc, err := s.GetRawSettings()
	if err == nil && doc.Content != "" {
		cfg, err := s.parseConfigFromContent(doc.Content)
		if err != nil {
			return model.NodeRedConfig{}, false, err
		}
		return cfg, true, nil
	}
	return model.NodeRedConfig{}, false, nil
}

func (s *ConfigService) backupSettingsFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	backupDir := filepath.Join(s.dataDir, "backups", "settings")
	// #nosec G304 -- path is the caller-validated settings.js inside the dataDir; write target is a derived backup file.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// The snapshot is created exclusively under a collision-resistant name,
	// so a save never overwrites the recovery point of another transaction
	// that landed in the same second (issue #883).
	return writeSettingsSnapshot(backupDir, time.Now().UTC(), data)
}

// renderSettings patches the live settings.js when one already exists, so
// operator-authored blocks (functionGlobalContext, externalModules, https,
// httpMiddleware, ...) survive a Save. Falls back to generating from scratch
// on first write or when the file is unreadable. Closes #577.
func (s *ConfigService) renderSettings(cfg model.NodeRedConfig) string {
	path := s.hostSvc.Detect().Settings.Path
	if path == "" {
		path = filepath.Join(s.dataDir, "settings.js")
	}
	// #nosec G304,G703 -- path is the configured settings.js path inside the dataDir; not request-derived.
	existing, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(existing))) == 0 {
		return generateSettingsJS(cfg)
	}
	return patchSettingsJS(string(existing), cfg)
}

// renderSettingsJS builds settings.js from scratch. Used for the default
// preview in GetRawSettings when no file exists; Save goes through
// (*ConfigService).renderSettings so existing content is patched, not clobbered.
func renderSettingsJS(cfg model.NodeRedConfig) string {
	return generateSettingsJS(cfg)
}

// generateSettingsJS builds settings.js from scratch (fallback for first-run)
func generateSettingsJS(cfg model.NodeRedConfig) string {
	var builder strings.Builder
	builder.WriteString("module.exports = {\n")
	fmt.Fprintf(&builder, "  uiPort: %d,\n", cfg.Port)
	if cfg.UIHost != "" {
		fmt.Fprintf(&builder, "  uiHost: %q,\n", cfg.UIHost)
	}
	fmt.Fprintf(&builder, "  httpAdminRoot: %q,\n", cfg.HTTPAdminRoot)
	fmt.Fprintf(&builder, "  httpNodeRoot: %q,\n", cfg.HTTPNodeRoot)
	if cfg.FlowFile != "" {
		fmt.Fprintf(&builder, "  flowFile: %q,\n", cfg.FlowFile)
	}
	if cfg.UserDir != "" {
		fmt.Fprintf(&builder, "  userDir: %q,\n", cfg.UserDir)
	}
	if cfg.NodesDir != "" {
		fmt.Fprintf(&builder, "  nodesDir: %q,\n", cfg.NodesDir)
	}
	fmt.Fprintf(&builder, "  disableEditor: %t,\n", cfg.DisableEditor)
	fmt.Fprintf(&builder, "  projectsEnabled: %t,\n", cfg.ProjectsEnabled)
	if cfg.Lang != "" {
		fmt.Fprintf(&builder, "  lang: %q,\n", cfg.Lang)
	}
	if block := renderEnvBlock(cfg); block != "" {
		builder.WriteString(block)
		builder.WriteString("\n")
	}
	if cfg.AdminAuth != nil && len(cfg.AdminAuth.Users) > 0 {
		builder.WriteString(renderAdminAuthBlock(cfg))
		builder.WriteString("\n")
	} else {
		builder.WriteString("  adminAuth: null,\n")
	}
	if cfg.HTTPNodeAuth != nil {
		builder.WriteString(renderHTTPBasicAuthBlock("httpNodeAuth", cfg.HTTPNodeAuth))
		builder.WriteString("\n")
	}
	if cfg.HTTPStaticAuth != nil {
		builder.WriteString(renderHTTPBasicAuthBlock("httpStaticAuth", cfg.HTTPStaticAuth))
		builder.WriteString("\n")
	}
	builder.WriteString("  editorTheme: {\n")
	fmt.Fprintf(&builder, "    projects: { enabled: %t },\n", cfg.ProjectsEnabled)
	builder.WriteString("  },\n")
	if block := renderLoggingBlock(loggingModelOrDefault(cfg.Logging)); block != "" {
		builder.WriteString(block)
		builder.WriteString("\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

// patchSettingsJS applies targeted regex substitutions to preserve unknown blocks
func patchSettingsJS(existingContent string, cfg model.NodeRedConfig) string {
	content := existingContent

	// Scalar keys: use line-level regex (handles comments, varying spacing)
	content = replaceScalarKey(content, "uiPort", fmt.Sprintf("%d", cfg.Port))
	if cfg.UIHost != "" {
		content = replaceScalarKey(content, "uiHost", fmt.Sprintf("%q", cfg.UIHost))
	}
	content = replaceScalarKey(content, "httpAdminRoot", fmt.Sprintf("%q", cfg.HTTPAdminRoot))
	content = replaceScalarKey(content, "httpNodeRoot", fmt.Sprintf("%q", cfg.HTTPNodeRoot))
	if cfg.FlowFile != "" {
		content = replaceScalarKey(content, "flowFile", fmt.Sprintf("%q", cfg.FlowFile))
	}
	if cfg.UserDir != "" {
		content = replaceScalarKey(content, "userDir", fmt.Sprintf("%q", cfg.UserDir))
	}
	if cfg.NodesDir != "" {
		content = replaceScalarKey(content, "nodesDir", fmt.Sprintf("%q", cfg.NodesDir))
	}
	content = replaceScalarKey(content, "disableEditor", fmt.Sprintf("%t", cfg.DisableEditor))
	content = replaceScalarKey(content, "projectsEnabled", fmt.Sprintf("%t", cfg.ProjectsEnabled))
	if cfg.Lang != "" {
		content = replaceScalarKey(content, "lang", fmt.Sprintf("%q", cfg.Lang))
	}

	// TLS / secrets (issue #762 slice 1). credentialSecret is treated as a
	// "rotate by saving a new value" operation: empty cfg leaves the
	// existing settings.js line untouched, "false" disables encryption,
	// any other value becomes the literal passphrase. The operator MUST
	// re-enter credentials after a rotation, so the UI is expected to
	// surface a confirmation warning before this key is written.
	if cfg.CredentialSecret != "" {
		if cfg.CredentialSecret == "false" {
			content = replaceScalarKey(content, "credentialSecret", "false")
		} else {
			content = replaceScalarKey(content, "credentialSecret", fmt.Sprintf("%q", cfg.CredentialSecret))
		}
	}
	content = replaceScalarKey(content, "requireHttps", fmt.Sprintf("%t", cfg.RequireHttps))
	if block := renderHttpsBlock(cfg); block != "" {
		content = replaceBlockKey(content, "https", block)
	} else {
		content = removeObjectKey(content, "https")
	}

	// Block keys: use brace-aware replacement
	if block := renderEnvBlock(cfg); block != "" {
		content = replaceArrayKey(content, "env", block)
	} else {
		content = removeArrayKey(content, "env")
	}
	if cfg.AdminAuth != nil {
		content = replaceBlockKey(content, "adminAuth", renderAdminAuthBlock(cfg))
	}
	if cfg.HTTPNodeAuth != nil {
		content = replaceBlockKey(content, "httpNodeAuth", renderHTTPBasicAuthBlock("httpNodeAuth", cfg.HTTPNodeAuth))
	}
	if cfg.HTTPStaticAuth != nil {
		content = replaceBlockKey(content, "httpStaticAuth", renderHTTPBasicAuthBlock("httpStaticAuth", cfg.HTTPStaticAuth))
	}
	content = replaceBlockKey(content, "editorTheme", renderEditorThemeBlock(cfg))
	content = patchLoggingSettings(content, cfg.Logging)
	content = ensureHTTPStatic(content, "uploads")

	return content
}

// ensureHTTPStatic makes Node-RED serve the configured uploads directory
// under its own origin so editorTheme image URLs (e.g. "/foo.png") resolve
// at http://<host>:1880/<filename>. Relative paths resolve against
// Node-RED's userDir; in the default all-in-one image, dataDir == userDir
// so "uploads" lands at <dataDir>/uploads. Idempotent: existing
// httpStatic (operator-authored, absolute or different relative path) is
// preserved — we only add the default when missing.
func ensureHTTPStatic(content, relDir string) string {
	re := regexp.MustCompile(`(?m)^\s*httpStatic\s*:`)
	if re.MatchString(content) {
		return content
	}
	return appendPropertyBeforeClosingBrace(content, fmt.Sprintf("  httpStatic: %q,\n", relDir))
}

// replaceScalarKey replaces or appends a scalar key:value pair
func replaceScalarKey(content, key, value string) string {
	// Try to match and replace existing line: key: value, (with optional trailing comma)
	rePattern := fmt.Sprintf(`(?m)^\s*%s\s*:\s*[^,\n]*,?\s*$`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(rePattern)

	if re.MatchString(content) {
		// Replace existing line, preserving indentation
		replacement := fmt.Sprintf("  %s: %s,", key, value)
		return re.ReplaceAllString(content, replacement)
	}

	// Key not found, append before closing brace
	return appendPropertyBeforeClosingBrace(content, fmt.Sprintf("  %s: %s,\n", key, value))
}

// appendPropertyBeforeClosingBrace inserts a rendered property just before
// the closing brace of the module.exports object.
//
// The property that precedes the insertion point must end with a comma. The
// Docker image ships a settings.js whose last property (`externalModules`)
// carries no trailing comma, and appending a key after it would render
// invalid JavaScript — every apply would then fail the candidate parse.
//
// A source that closes the object literal with a semicolon (`};`) is
// supported too: the semicolon terminates the statement, not the object, so
// it is kept after the closing brace. Without it the insertion point is not
// found and the rendered property is dropped — the apply then reports a
// success while the value never reaches settings.js (issue #882).
func appendPropertyBeforeClosingBrace(content, entry string) string {
	appendPattern := `\n\s*\}(;?)\s*$`
	appendRe := regexp.MustCompile(appendPattern)
	loc := appendRe.FindStringSubmatchIndex(content)
	if loc == nil {
		return content
	}
	// The content is concatenated rather than passed through
	// ReplaceAllString: the rendered entry can carry a bcrypt hash whose
	// `$` characters ReplaceAllString would interpret as a capture
	// reference.
	head := ensureCommaTerminated(content[:loc[0]])
	semicolon := ""
	if loc[2] >= 0 {
		semicolon = ";"
	}
	return head + "\n" + entry + "}" + semicolon + "\n"
}

// ensureCommaTerminated appends a comma to the last property of the content
// when it does not carry one. Content that ends with an opening delimiter
// (an empty object or array) needs no comma.
func ensureCommaTerminated(content string) string {
	trimmed := strings.TrimRight(content, " \t")
	if trimmed == "" {
		return content
	}
	switch trimmed[len(trimmed)-1] {
	case ',', '{', '[':
		return content
	}
	return trimmed + "," + content[len(trimmed):]

}

// replaceArrayKey replaces or appends a multi-line array key.
func replaceArrayKey(content, key, arrayContent string) string {
	rePattern := fmt.Sprintf(`(?ms)^\s*%s\s*:\s*\[[^\]]*\],?\s*$`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(rePattern)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, arrayContent)
	}

	return appendPropertyBeforeClosingBrace(content, arrayContent+"\n")
}

func removeArrayKey(content, key string) string {
	rePattern := fmt.Sprintf(`(?ms)^\s*%s\s*:\s*\[[^\]]*\],?\s*\n?`, regexp.QuoteMeta(key))
	return regexp.MustCompile(rePattern).ReplaceAllString(content, "")
}

// removeObjectKey deletes a top-level object key from a settings.js
// document. The matching brace-counting helper (findTopLevelBlock) was
// introduced for editorTheme in #748; we reuse the same approach here so
// a `https: { ... }` block is removed verbatim — including any inner
// nested braces — without leaving a dangling `https: null` behind.
func removeObjectKey(content, key string) string {
	start, end, ok := findTopLevelBlock(content, key)
	if !ok {
		return content
	}
	return content[:start] + content[end:]
}

// renderHttpsBlock produces a settings.js-compatible `https: { ... }` entry
// from the structured HttpsConfig. Each non-empty field becomes an
// fs.readFileSync(<path>) call so Node-RED can consume on-disk PEM
// material without us embedding certificate bytes in nrcc's own JSON
// store. Returns the empty string when no fields are set, so callers
// route to removeObjectKey instead of emitting an empty block.
func renderHttpsBlock(cfg model.NodeRedConfig) string {
	if cfg.Https == nil {
		return ""
	}
	parts := make([]string, 0, 5)
	if cfg.Https.Key != "" {
		parts = append(parts, fmt.Sprintf("    key: require('fs').readFileSync(%q),", cfg.Https.Key))
	}
	if cfg.Https.Cert != "" {
		parts = append(parts, fmt.Sprintf("    cert: require('fs').readFileSync(%q),", cfg.Https.Cert))
	}
	if cfg.Https.CA != "" {
		parts = append(parts, fmt.Sprintf("    ca: require('fs').readFileSync(%q),", cfg.Https.CA))
	}
	if cfg.Https.Port > 0 {
		parts = append(parts, fmt.Sprintf("    port: %d,", cfg.Https.Port))
	}
	if cfg.Https.Passphrase != "" {
		parts = append(parts, fmt.Sprintf("    passphrase: %q,", cfg.Https.Passphrase))
	}
	if len(parts) == 0 {
		return ""
	}
	return "  https: {\n" + strings.Join(parts, "\n") + "\n  },"
}

func nodeRedEnvType(typ string) string {
	switch typ {
	case "number":
		return "num"
	case "boolean":
		return "bool"
	default:
		return "str"
	}
}

func renderEnvBlock(cfg model.NodeRedConfig) string {
	vars := make([]model.EnvVar, 0, len(cfg.EnvVars))
	for _, envVar := range cfg.EnvVars {
		typ := envVar.Type
		if typ == "" || typ == "plain" {
			typ = "string"
		}
		if envVar.Key == "" || envVar.Encrypted || typ == "secret" {
			continue
		}
		vars = append(vars, model.EnvVar{Key: envVar.Key, Value: envVar.Value, Type: nodeRedEnvType(typ)})
	}
	if len(vars) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("  env: [\n")
	for i, envVar := range vars {
		builder.WriteString("    {\n")
		fmt.Fprintf(&builder, "      name: %q,\n", envVar.Key)
		fmt.Fprintf(&builder, "      value: %q,\n", envVar.Value)
		fmt.Fprintf(&builder, "      type: %q\n", envVar.Type)
		if i == len(vars)-1 {
			builder.WriteString("    }\n")
		} else {
			builder.WriteString("    },\n")
		}
	}
	builder.WriteString("  ],")
	return builder.String()
}

// renderAdminAuthBlock renders adminAuth as a block string
func renderAdminAuthBlock(cfg model.NodeRedConfig) string {
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) == 0 {
		return "  adminAuth: null,"
	}

	var builder strings.Builder
	builder.WriteString("  adminAuth: {\n")
	fmt.Fprintf(&builder, "    type: %q,\n", cfg.AdminAuth.Type)
	builder.WriteString("    users: [\n")
	for i, user := range cfg.AdminAuth.Users {
		builder.WriteString("      {\n")
		fmt.Fprintf(&builder, "        username: %q,\n", user.Username)
		fmt.Fprintf(&builder, "        password: %q,\n", user.Password)
		fmt.Fprintf(&builder, "        permissions: %q\n", user.Permissions)
		if i == len(cfg.AdminAuth.Users)-1 {
			builder.WriteString("      }\n")
		} else {
			builder.WriteString("      },\n")
		}
	}
	builder.WriteString("    ],\n")
	if cfg.AdminAuth.SessionExpiryTime > 0 {
		fmt.Fprintf(&builder, "    sessionExpiryTime: %d,\n", cfg.AdminAuth.SessionExpiryTime)
	}
	builder.WriteString("  },")
	return builder.String()
}

func renderHTTPBasicAuthBlock(name string, auth *model.HTTPBasicAuth) string {
	if auth == nil {
		return ""
	}
	return fmt.Sprintf("  %s: { user: %q, pass: %q },", name, auth.User, auth.Pass)
}

// renderEditorThemeBlock renders editorTheme as a block string. The block is
// reconstructed from `cfg.EditorTheme` (interface{} from the JSON store), so
// every sub-key the frontend sends — page, header, palette, codeEditor,
// deployButton, login, logout, userMenu, tours, projects — round-trips into
// settings.js. Previously this function hardcoded `projects.enabled` and
// dropped everything else, which is why #707 reported that Editor Theme
// changes never propagated to the live Node-RED config.
//
// The encoder is intentionally lenient: keys with non-string / non-primitive
// values are skipped (we never want to inject unparseable JS into
// settings.js), and unset sub-objects are omitted entirely so the output
// stays minimal.
func renderEditorThemeBlock(cfg model.NodeRedConfig) string {
	theme, _ := cfg.EditorTheme.(map[string]any)
	if len(theme) == 0 {
		// Fall back to the historical minimal block when nothing is set,
		// so first-write settings.js still emits a syntactically valid
		// editorTheme entry.
		return fmt.Sprintf("  editorTheme: {\n    projects: { enabled: %t },\n  },", cfg.ProjectsEnabled)
	}

	quote := func(s string) string { return fmt.Sprintf("%q", s) }
	boolLit := func(v any) (string, bool) {
		b, ok := v.(bool)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("%t", b), true
	}
	intLit := func(v any) (string, bool) {
		switch n := v.(type) {
		case int:
			return fmt.Sprintf("%d", n), true
		case int64:
			return fmt.Sprintf("%d", n), true
		case float64:
			return fmt.Sprintf("%d", int(n)), true
		}
		return "", false
	}
	stringLit := func(v any) (string, bool) {
		s, ok := v.(string)
		if !ok {
			return "", false
		}
		return quote(s), true
	}

	var builder strings.Builder
	builder.WriteString("  editorTheme: {\n")

	// page { title, favicon, css }
	if page, ok := theme["page"].(map[string]any); ok {
		parts := []string{}
		if s, ok := stringLit(page["title"]); ok {
			parts = append(parts, fmt.Sprintf("title: %s", s))
		}
		if s, ok := stringLit(page["favicon"]); ok {
			parts = append(parts, fmt.Sprintf("favicon: %s", s))
		}
		// css can be a string OR an array of strings (Node-RED accepts both)
		switch css := page["css"].(type) {
		case string:
			if css != "" {
				parts = append(parts, fmt.Sprintf("css: %s", quote(css)))
			}
		case []any:
			quoted := []string{}
			for _, item := range css {
				if s, ok := item.(string); ok {
					quoted = append(quoted, quote(s))
				}
			}
			if len(quoted) > 0 {
				parts = append(parts, "css: ["+strings.Join(quoted, ", ")+"]")
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(&builder, "    page: { %s },\n", strings.Join(parts, ", "))
		}
	}

	// header { title, image, url }
	if header, ok := theme["header"].(map[string]any); ok {
		parts := []string{}
		if s, ok := stringLit(header["title"]); ok {
			parts = append(parts, fmt.Sprintf("title: %s", s))
		}
		if s, ok := stringLit(header["image"]); ok {
			parts = append(parts, fmt.Sprintf("image: %s", s))
		}
		if s, ok := stringLit(header["url"]); ok {
			parts = append(parts, fmt.Sprintf("url: %s", s))
		}
		if len(parts) > 0 {
			fmt.Fprintf(&builder, "    header: { %s },\n", strings.Join(parts, ", "))
		}
	}

	// deployButton { type, label, icon }
	if db, ok := theme["deployButton"].(map[string]any); ok {
		parts := []string{}
		if s, ok := stringLit(db["type"]); ok {
			parts = append(parts, fmt.Sprintf("type: %s", s))
		}
		if s, ok := stringLit(db["label"]); ok {
			parts = append(parts, fmt.Sprintf("label: %s", s))
		}
		if s, ok := stringLit(db["icon"]); ok {
			parts = append(parts, fmt.Sprintf("icon: %s", s))
		}
		if len(parts) > 0 {
			fmt.Fprintf(&builder, "    deployButton: { %s },\n", strings.Join(parts, ", "))
		}
	}

	// login { image }
	if login, ok := theme["login"].(map[string]any); ok {
		if s, ok := stringLit(login["image"]); ok && s != `""` {
			fmt.Fprintf(&builder, "    login: { image: %s },\n", s)
		}
	}

	// logout { redirect }
	if logout, ok := theme["logout"].(map[string]any); ok {
		if s, ok := stringLit(logout["redirect"]); ok && s != `""` {
			fmt.Fprintf(&builder, "    logout: { redirect: %s },\n", s)
		}
	}

	// palette { editable, catalogues }
	if palette, ok := theme["palette"].(map[string]any); ok {
		parts := []string{}
		if b, ok := boolLit(palette["editable"]); ok {
			parts = append(parts, fmt.Sprintf("editable: %s", b))
		}
		switch cats := palette["catalogues"].(type) {
		case []any:
			quoted := []string{}
			for _, item := range cats {
				if s, ok := item.(string); ok {
					quoted = append(quoted, quote(s))
				}
			}
			if len(quoted) > 0 {
				parts = append(parts, "catalogues: ["+strings.Join(quoted, ", ")+"]")
			}
		case string:
			if cats != "" {
				parts = append(parts, fmt.Sprintf("catalogues: %s", quote(cats)))
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(&builder, "    palette: { %s },\n", strings.Join(parts, ", "))
		}
	}

	// projects { enabled } — also reflected from cfg.ProjectsEnabled for
	// backward compatibility with configs that don't carry an explicit
	// editorTheme.projects block.
	projectsEnabled := cfg.ProjectsEnabled
	if projects, ok := theme["projects"].(map[string]any); ok {
		if b, ok := boolLit(projects["enabled"]); ok {
			projectsEnabled = b == "true"
		}
	}
	fmt.Fprintf(&builder, "    projects: { enabled: %t },\n", projectsEnabled)

	// codeEditor { lib, options { theme, fontSize } }
	if ce, ok := theme["codeEditor"].(map[string]any); ok {
		parts := []string{}
		if s, ok := stringLit(ce["lib"]); ok {
			parts = append(parts, fmt.Sprintf("lib: %s", s))
		}
		if opts, ok := ce["options"].(map[string]any); ok {
			optParts := []string{}
			if s, ok := stringLit(opts["theme"]); ok {
				optParts = append(optParts, fmt.Sprintf("theme: %s", s))
			}
			if n, ok := intLit(opts["fontSize"]); ok {
				optParts = append(optParts, fmt.Sprintf("fontSize: %s", n))
			}
			if len(optParts) > 0 {
				parts = append(parts, "options: { "+strings.Join(optParts, ", ")+" }")
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(&builder, "    codeEditor: { %s },\n", strings.Join(parts, ", "))
		}
	}

	// userMenu and tours are booleans; render only when explicitly set so
	// we don't override existing settings.js values with implicit defaults.
	if um, ok := boolLit(theme["userMenu"]); ok {
		fmt.Fprintf(&builder, "    userMenu: %s,\n", um)
	}
	if tr, ok := boolLit(theme["tours"]); ok {
		fmt.Fprintf(&builder, "    tours: %s,\n", tr)
	}

	builder.WriteString("  },")
	return builder.String()
}

// loggingModelOrDefault normalises the logging value of the structured
// model and falls back to Node-RED's default console handler when the
// operator has not configured one.
func loggingModelOrDefault(value any) loggingModel {
	logging := newLoggingModel(value)
	if len(logging) == 0 {
		return defaultLoggingModel()
	}
	return logging
}

func (s *ConfigService) parseConfigFromContent(content string) (model.NodeRedConfig, error) {
	cfg := model.DefaultNodeRedConfig()
	cfg.Port = parseIntFromJS(content, "uiPort", parseIntFromJS(content, "port", cfg.Port))
	cfg.UIPort = cfg.Port
	cfg.UIHost = parseStringFromJS(content, "uiHost", cfg.UIHost)
	cfg.HTTPAdminRoot = parseStringFromJS(content, "httpAdminRoot", cfg.HTTPAdminRoot)
	cfg.HTTPNodeRoot = parseStringFromJS(content, "httpNodeRoot", cfg.HTTPNodeRoot)
	cfg.FlowFile = parseStringFromJS(content, "flowFile", cfg.FlowFile)
	cfg.UserDir = parseStringFromJS(content, "userDir", cfg.UserDir)
	cfg.NodesDir = parseStringFromJS(content, "nodesDir", cfg.NodesDir)
	cfg.Lang = parseStringFromJS(content, "lang", cfg.Lang)
	cfg.DisableEditor = parseBoolFromJS(content, "disableEditor", cfg.DisableEditor)
	cfg.ProjectsEnabled = parseProjectsEnabledFromJS(content, cfg.ProjectsEnabled)
	cfg.CredentialSecret = parseCredentialSecretFromJS(content)
	cfg.RequireHttps = parseBoolFromJS(content, "requireHttps", false)
	cfg.Https = parseHttpsBlockFromJS(content)
	if logging := parseLoggingBlock(content); len(logging) > 0 {
		// The logging block is read back from the source so the mirror
		// reports the values Node-RED actually runs with (issue #882).
		cfg.Logging = logging.value()
	}
	surfaces, err := parseAuthenticationSurfacesFromJS(content)
	if err != nil {
		// Inspection failures are validation failures, not missing auth.
		return cfg, err
	}
	if surfaces != nil {
		cfg.AdminAuth = surfaces.AdminAuth
		cfg.HTTPNodeAuth = surfaces.HTTPNodeAuth
		cfg.HTTPStaticAuth = surfaces.HTTPStaticAuth
	}
	return cfg, nil
}

// parseCredentialSecretFromJS reads the credentialSecret scalar back out of
// settings.js. The value can be a quoted string (operator-supplied
// passphrase) or the literal `false` (encryption disabled). Anything else
// — including an empty string — is treated as "unset" so the UI doesn't
// display a stale value.
func parseCredentialSecretFromJS(content string) string {
	re := regexp.MustCompile(`credentialSecret\s*:\s*(?:'([^']*)'|"([^"]*)")`)
	if m := re.FindStringSubmatch(content); len(m) > 0 {
		if m[1] != "" {
			return m[1]
		}
		return m[2]
	}
	if regexp.MustCompile(`credentialSecret\s*:\s*false`).MatchString(content) {
		return "false"
	}
	return ""
}

// parseHttpsBlockFromJS locates the top-level `https: { ... }` block using
// the same brace-counting helper as the renderer, then extracts the
// fs.readFileSync(<path>) calls and quoted scalars into a HttpsConfig.
// Returns nil when no https block is present so the structured model
// reflects "TLS not configured" rather than an empty placeholder.
func parseHttpsBlockFromJS(content string) *model.HttpsConfig {
	start, end, ok := findTopLevelBlock(content, "https")
	if !ok {
		return nil
	}
	body := content[start:end]

	readRe := regexp.MustCompile(`(\w+)\s*:\s*require\(['"]fs['"]\)\.readFileSync\(['"]([^'"]+)['"]\)`)
	quoteRe := regexp.MustCompile(`(\w+)\s*:\s*['"]([^'"]+)['"]`)
	intRe := regexp.MustCompile(`port\s*:\s*(\d+)`)

	https := &model.HttpsConfig{}
	for _, m := range readRe.FindAllStringSubmatch(body, -1) {
		switch m[1] {
		case "key":
			https.Key = m[2]
		case "cert":
			https.Cert = m[2]
		case "ca":
			https.CA = m[2]
		}
	}
	for _, m := range quoteRe.FindAllStringSubmatch(body, -1) {
		if m[1] == "passphrase" {
			https.Passphrase = m[2]
		}
	}
	if m := intRe.FindStringSubmatch(body); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			https.Port = n
		}
	}
	if https.Key == "" && https.Cert == "" && https.CA == "" && https.Port == 0 && https.Passphrase == "" {
		return nil
	}
	return https
}

func parseStringFromJS(content, key, fallback string) string {
	re := regexp.MustCompile(key + `\s*:\s*['"]([^'"]+)['"]`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1]
	}
	return fallback
}

// parseProjectsEnabledFromJS looks for projectsEnabled key specifically (not just "enabled")
// This avoids matching "enabled:" inside logging.console.metrics or other nested properties
func parseProjectsEnabledFromJS(content string, fallback bool) bool {
	// First try to find standalone "projectsEnabled: true|false"
	re := regexp.MustCompile(`projectsEnabled\s*:\s*(true|false)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1] == "true"
	}

	// Fallback to editorTheme.projects.enabled pattern (nested)
	// This handles: editorTheme: { projects: { enabled: true } }
	nestedRe := regexp.MustCompile(`editorTheme\s*:\s*\{[^}]*projects\s*:\s*\{[^}]*enabled\s*:\s*(true|false)`)
	if nestedMatches := nestedRe.FindStringSubmatch(content); len(nestedMatches) == 2 {
		return nestedMatches[1] == "true"
	}

	return fallback
}

// parseAdminAuthFromJS extracts the adminAuth block from raw settings.js content
// using a goja-based JavaScript sandbox (see settings_sandbox.go). The legacy
// regex implementation was removed because it failed on escaped quotes, line
// comments, block comments and silently picked the first adminAuth block when
// several were present. The sandbox observes actual script execution order,
// so the value returned is whatever the script actually exported.
//
// Returns nil when the script does not export an adminAuth block or the
// content cannot be parsed safely; preserves the legacy contract that callers
// (parseConfigFromContent, LoadSettings) can simply check for nil. Sandbox
// timeouts (ErrSandboxTimeout) are propagated so the HTTP layer can map them
// to a 4xx response instead of silently dropping adminAuth.
func parseAdminAuthFromJS(content string) (*model.AdminAuth, error) {
	auth, err := ParseAdminAuthViaSandbox(content)
	if err != nil {
		// ErrSandboxTimeout must reach the handler so the operator gets a
		// clear "settings.js did not terminate" instead of a hung request.
		// Other sandbox errors (forbidden host APIs, syntax) keep the
		// legacy contract of returning nil auth.
		if errors.Is(err, ErrSandboxTimeout) {
			return nil, err
		}
		return nil, nil
	}
	return auth, nil
}

func parseAuthenticationSurfacesFromJS(content string) (*model.AuthenticationSurfaces, error) {
	surfaces, err := ParseAuthenticationSurfacesViaSandbox(content)
	if err != nil {
		if errors.Is(err, ErrAdminAuthMissing) || errors.Is(err, ErrSandboxHostAPI) {
			// A successful inspection without auth and source-managed settings
			// requiring Node-RED host APIs are distinct sandbox outcomes. The
			// latter retain their source unchanged; auth cannot be inspected.
			return nil, nil
		}
		return nil, err
	}
	if surfaces.AdminAuth != nil && surfaces.AdminAuth.Type != "credentials" {
		// Non-credential strategies can contain executable callbacks. They are
		// intentionally source-managed until a dedicated strategy editor exists.
		surfaces.AdminAuth = nil
	}
	return surfaces, nil
}

func parseIntFromJS(content, key string, fallback int) int {
	re := regexp.MustCompile(key + `\s*:\s*(\d+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		value, err := strconv.Atoi(matches[1])
		if err == nil {
			return value
		}
	}
	return fallback
}

func parseBoolFromJS(content, key string, fallback bool) bool {
	re := regexp.MustCompile(key + `\s*:\s*(true|false)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1] == "true"
	}
	return fallback
}

// findTopLevelBlock scans content for `key:` at the start of a line
// (preceded only by whitespace) and returns the byte range [start, end)
// of the full entry — block or scalar — including its leading
// indentation and trailing comma. Returns ok=false if the key is not
// present. start points to the first byte of the line's leading
// whitespace so the caller can substitute any replacement verbatim.
//
// This is brace-counting, not regex. The previous regex couldn't match
// blocks deeper than one level of nesting — which broke editorTheme as
// soon as the user had `codeEditor.options.theme` set, because the
// existing-block replacement failed and the function fell through to
// "append a second block", producing duplicates on every Save (#723).
func findTopLevelBlock(content, key string) (start, end int, ok bool) {
	prefix := key + ":"
	scanFrom := 0
	for {
		lineStart, keyStart := indexOfKeyAtLineStart(content, prefix, scanFrom)
		if keyStart < 0 {
			return 0, 0, false
		}
		// Position right after the colon.
		after := keyStart + len(prefix)
		// Skip whitespace (space, tab) but not newlines.
		for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
			after++
		}
		if after >= len(content) {
			return 0, 0, false
		}
		switch content[after] {
		case '{':
			_, endIdx := findMatchingBrace(content, after)
			if endIdx < 0 {
				return 0, 0, false
			}
			// Consume any trailing comma and the rest of the line.
			endIdx = consumeLineEnd(content, endIdx)
			return lineStart, endIdx, true
		default:
			// Scalar value (null, true, "string", etc.) — cover the rest of the line.
			lineEnd := strings.IndexByte(content[after:], '\n')
			if lineEnd < 0 {
				return lineStart, len(content), true
			}
			return lineStart, after + lineEnd, true
		}
	}
}

// indexOfKeyAtLineStart returns the start of the line and the start of
// the matched needle. The line start is the position just after the
// previous newline (or 0), so the caller can substitute the entire
// indented line. The key start is where the needle itself begins. -1, -1
// is returned if no match is found at or after `from`. This replaces the
// regex `(?:^|\n)\s*` constraint without pulling in the regexp engine.
func indexOfKeyAtLineStart(content, needle string, from int) (lineStart, keyStart int) {
	atLineStart := from == 0
	if atLineStart && len(content) > 0 && (content[0] == ' ' || content[0] == '\t') {
		atLineStart = false
	}
	for i := from; i+len(needle) <= len(content); i++ {
		if atLineStart {
			j := i
			for j < len(content) && (content[j] == ' ' || content[j] == '\t') {
				j++
			}
			if j+len(needle) <= len(content) && content[j:j+len(needle)] == needle {
				return i, j
			}
			atLineStart = false
		}
		if content[i] == '\n' {
			atLineStart = true
		}
	}
	return -1, -1
}

// findMatchingBrace returns the index of the '}' that matches the '{' at
// openIdx, plus the index just past it. Handles nested braces, string
// literals (', ", `) and line/block comments. Returns -1 if unmatched.
func findMatchingBrace(content string, openIdx int) (closeIdx, afterClose int) {
	depth := 0
	inString := byte(0)
	for i := openIdx; i < len(content); i++ {
		c := content[i]
		if inString != 0 {
			if c == '\\' && i+1 < len(content) {
				i++
				continue
			}
			if c == inString {
				inString = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			inString = c
		case '/':
			// line comment "//"
			if i+1 < len(content) && content[i+1] == '/' {
				nl := strings.IndexByte(content[i:], '\n')
				if nl < 0 {
					return -1, -1
				}
				i += nl
				continue
			}
			// block comment /* */
			if i+1 < len(content) && content[i+1] == '*' {
				end := strings.Index(content[i:], "*/")
				if end < 0 {
					return -1, -1
				}
				i += end + 1
				continue
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, i + 1
			}
		}
	}
	return -1, -1
}

// consumeLineEnd skips a trailing comma and any trailing whitespace on
// the same line. It deliberately does NOT consume the newline itself —
// the caller will substitute the replacement block content (which has no
// trailing newline) and we want the original newline to remain in the
// document so the next line keeps its leading context.
func consumeLineEnd(content string, endIdx int) int {
	for endIdx < len(content) && (content[endIdx] == ' ' || content[endIdx] == '\t' || content[endIdx] == ',') {
		endIdx++
	}
	return endIdx
}

func replaceBlockKey(content, key, blockContent string) string {
	start, end, ok := findTopLevelBlock(content, key)
	if !ok {
		// Append before closing brace of the module.exports object.
		// blockContent already includes its own leading indent and trailing comma.
		return appendPropertyBeforeClosingBrace(content, blockContent+"\n")
	}
	return content[:start] + blockContent + content[end:]
}

// (helper for tests)
