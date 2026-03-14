package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	spec3 "github.com/shuttlefy/go-openapi3-spec"
)

func Write(doc *spec3.OpenAPI, format, path string) error {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(doc, "", "  ")
	case "yaml":
		data, err = spec3.MarshalYAML(doc)
	default:
		return fmt.Errorf("unsupported output format: %q (use json or yaml)", format)
	}
	if err != nil {
		return fmt.Errorf("marshal %s: %w", format, err)
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
