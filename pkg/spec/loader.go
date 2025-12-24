// Package spec defines the cluster specification schema for kspec.
package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads a cluster specification from a YAML file.
func LoadFromFile(path string) (*ClusterSpecification, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file %s: %w", path, err)
	}

	var spec ClusterSpecification
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec file %s: %w", path, err)
	}

	return &spec, nil
}
