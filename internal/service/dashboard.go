package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
)

const (
	discoveryAvailable  = "available"
	discoveryAbsent     = "absent"
	discoveryMalformed  = "malformed"
	discoveryUnknown    = "unknown"
	legacyDashboard     = "node-red-dashboard"
	legacyDashboardPath = "/ui"
	flowFuseDashboard   = "@flowfuse/node-red-dashboard"
)

// DashboardService discovers dashboard capabilities without changing Node-RED data.
type DashboardService struct {
	dataDir string
}

// NewDashboardService creates a read-only dashboard discovery service.
func NewDashboardService(dataDir string) *DashboardService {
	return &DashboardService{dataDir: dataDir}
}

// Discover returns the installed dashboard packages and FlowFuse ui-base paths.
func (s *DashboardService) Discover() model.DashboardDiscovery {
	result := model.DashboardDiscovery{
		Packages: model.DiscoverySource{State: discoveryAbsent},
		Flows:    model.DiscoverySource{State: discoveryAbsent},
		FlowFuse: make([]model.DashboardPackage, 0),
		UIBases:  make([]model.DashboardPath, 0),
	}
	s.discoverPackages(&result)
	s.discoverUIBases(&result)
	return result
}

func (s *DashboardService) discoverPackages(result *model.DashboardDiscovery) {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "package.json")) // #nosec G304 -- dataDir is operator-supplied.
	if err != nil {
		if !os.IsNotExist(err) {
			result.Packages.State = discoveryUnknown
		}
		return
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		result.Packages.State = discoveryMalformed
		return
	}
	result.Packages.State = discoveryAvailable
	if version, ok := pkg.Dependencies[legacyDashboard]; ok {
		result.Legacy = &model.LegacyDashboard{
			Package: model.DashboardPackage{Name: legacyDashboard, Version: version},
			Path:    legacyDashboardPath,
		}
	}
	if version, ok := pkg.Dependencies[flowFuseDashboard]; ok {
		result.FlowFuse = append(result.FlowFuse, model.DashboardPackage{Name: flowFuseDashboard, Version: version})
	}
}

func (s *DashboardService) discoverUIBases(result *model.DashboardDiscovery) {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "flows.json")) // #nosec G304 -- dataDir is operator-supplied.
	if err != nil {
		if !os.IsNotExist(err) {
			result.Flows.State = discoveryUnknown
		}
		return
	}
	var flows []map[string]interface{}
	if err := json.Unmarshal(data, &flows); err != nil {
		result.Flows.State = discoveryMalformed
		return
	}
	result.Flows.State = discoveryAvailable
	for _, node := range flows {
		if stringField(node, "type") != "ui-base" {
			continue
		}
		path := strings.TrimSpace(stringField(node, "path"))
		state := discoveryUnknown
		if path != "" {
			state = discoveryAvailable
		}
		result.UIBases = append(result.UIBases, model.DashboardPath{
			NodeID: stringField(node, "id"), Path: path, State: state,
		})
	}
}
