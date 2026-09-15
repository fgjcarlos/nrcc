package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const (
	dashboardTargetLegacy   = "legacy"
	dashboardTargetFlowFuse = "flowfuse"
	dashboardRecipeBasic    = "basic-auth"
	dashboardRecipeProxySSO = "reverse-proxy-sso"
)

var ErrDashboardPolicyUnavailable = errors.New("dashboard discovery is absent or ambiguous")
var ErrUnsafeDashboardPolicy = errors.New("unsupported dashboard access policy")

// DashboardAccessService renders reviewed recipes and applies them transactionally.
type DashboardAccessService struct {
	discovery *DashboardService
	config    *ConfigService
	apply     *ApplyCoordinator
}

func NewDashboardAccessService(discovery *DashboardService, config *ConfigService, apply *ApplyCoordinator) *DashboardAccessService {
	return &DashboardAccessService{discovery: discovery, config: config, apply: apply}
}

func (s *DashboardAccessService) Apply(ctx context.Context, policy model.DashboardAccessPolicy, actor string, request *http.Request) (model.SettingsDocument, error) {
	if s.discovery == nil || s.config == nil || s.apply == nil || !dashboardPolicySupported(s.discovery.Discover(), policy) {
		return model.SettingsDocument{}, ErrDashboardPolicyUnavailable
	}
	if err := validateDashboardPolicy(policy); err != nil {
		return model.SettingsDocument{}, err
	}
	live, err := s.config.GetRawSettings()
	if err != nil {
		return model.SettingsDocument{}, fmt.Errorf("read dashboard settings: %w", err)
	}
	content, err := renderDashboardPolicy(live.Content, policy)
	if err != nil {
		return model.SettingsDocument{}, err
	}
	expected := live.Revision
	if policy.ExpectedRevision != "" {
		expected = model.SourceRevision{Fingerprint: policy.ExpectedRevision, Algorithm: SourceRevisionAlgorithm}
	}
	_, err = s.apply.Apply(ctx, ApplyRequest{
		Path: live.Path, Content: content, Expected: expected,
		BackupDir: filepath.Join(s.config.dataDir, "backups", "settings"), Actor: actor, Request: request,
		Capabilities: s.config.ConfigurationCapabilities(),
	})
	if err != nil {
		return model.SettingsDocument{}, err
	}
	return s.config.GetRawSettings()
}

func dashboardPolicySupported(discovery model.DashboardDiscovery, policy model.DashboardAccessPolicy) bool {
	if discovery.Packages.State != discoveryAvailable {
		return false
	}
	switch policy.Target {
	case dashboardTargetLegacy:
		return discovery.Legacy != nil
	case dashboardTargetFlowFuse:
		if len(discovery.FlowFuse) == 0 || discovery.Flows.State != discoveryAvailable || len(discovery.UIBases) == 0 {
			return false
		}
		for _, base := range discovery.UIBases {
			if base.State != discoveryAvailable || base.Path == "" {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validateDashboardPolicy(policy model.DashboardAccessPolicy) error {
	if policy.Recipe != dashboardRecipeBasic && policy.Recipe != dashboardRecipeProxySSO {
		return ErrUnsafeDashboardPolicy
	}
	if strings.TrimSpace(policy.Secret) == "" || strings.ContainsAny(policy.Secret, "\r\n") {
		return ErrUnsafeDashboardPolicy
	}
	if policy.Recipe == dashboardRecipeBasic && (strings.TrimSpace(policy.Username) == "" || strings.Contains(policy.Username, ":")) {
		return ErrUnsafeDashboardPolicy
	}
	return nil
}

func renderDashboardPolicy(source string, policy model.DashboardAccessPolicy) (string, error) {
	if policy.Target == dashboardTargetLegacy {
		if policy.Recipe != dashboardRecipeBasic {
			return "", ErrUnsafeDashboardPolicy
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(policy.Secret), BcryptCost)
		if err != nil {
			return "", ErrUnsafeDashboardPolicy
		}
		block := fmt.Sprintf("  httpNodeAuth: { user: %s, pass: %s },", strconv.Quote(policy.Username), strconv.Quote(string(hash)))
		patched, err := SourcePatch(source, []SourceEdit{{Key: "httpNodeAuth", Block: block, IsBlock: true}})
		return patched.Content, err
	}
	if _, _, exists := findTopLevelBlock(source, "dashboard"); exists {
		return "", ErrDashboardPolicyUnavailable
	}
	value := policy.Secret
	if policy.Recipe == dashboardRecipeBasic {
		value = "Basic " + base64.StdEncoding.EncodeToString([]byte(policy.Username+":"+policy.Secret))
	}
	quoted := strconv.Quote(value)
	header := "authorization"
	if policy.Recipe == dashboardRecipeProxySSO {
		header = "x-nrcc-dashboard-sso"
	}
	block := fmt.Sprintf(`  dashboard: {
    middleware: (req, res, next) => { if (req.headers[%q] === %s) return next(); res.status(401).end(); },
    ioMiddleware: (socket, next) => { if (socket.request.headers[%q] === %s) return next(); next(new Error("unauthorized")); }
  },`, header, quoted, header, quoted)
	patched, err := SourcePatch(source, []SourceEdit{{Key: "dashboard", Block: block, IsBlock: true}})
	return patched.Content, err
}
