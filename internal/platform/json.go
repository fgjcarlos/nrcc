package platform

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read json file: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode json file: %w", err)
	}
	return nil
}
