package service

import (
	"nrcc/internal/model"
)

// DefaultFullAppConfig is a convenience wrapper that delegates to model.DefaultFullAppConfig()
func DefaultFullAppConfig() model.FullAppConfig {
	return model.DefaultFullAppConfig()
}
