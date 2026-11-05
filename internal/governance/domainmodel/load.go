package domainmodel

import (
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

func LoadEndpointInventory(path string) (*EndpointInventory, error) {
	var value EndpointInventory
	if err := loadStrict(path, &value); err != nil {
		return nil, fmt.Errorf("load endpoint inventory: %w", err)
	}
	if err := validateEndpointInventory(&value); err != nil {
		return nil, err
	}
	return &value, nil
}

func LoadDomainModel(path string) (*DomainModel, error) {
	var value DomainModel
	if err := loadStrict(path, &value); err != nil {
		return nil, fmt.Errorf("load domain model: %w", err)
	}
	if err := validateDomainModel(&value); err != nil {
		return nil, err
	}
	return &value, nil
}

func LoadDependencyCapabilities(path string) (*DependencyCapabilities, error) {
	var value DependencyCapabilities
	if err := loadStrict(path, &value); err != nil {
		return nil, fmt.Errorf("load dependency capabilities: %w", err)
	}
	if err := validateDependencyCapabilities(&value); err != nil {
		return nil, err
	}
	return &value, nil
}

func LoadCapabilityMap(path string) (*CapabilityMap, error) {
	var value CapabilityMap
	if err := loadStrict(path, &value); err != nil {
		return nil, fmt.Errorf("load capability map: %w", err)
	}
	if err := validateCapabilityMap(&value); err != nil {
		return nil, err
	}
	return &value, nil
}

func loadStrict(path string, destination any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple YAML documents are not allowed")
		}
		return fmt.Errorf("decode trailing YAML: %w", err)
	}
	return nil
}

func requireSchemaVersion(version, expected int, manifest string) error {
	if version != expected {
		return fmt.Errorf("%s.schema_version = %d, want %d", manifest, version, expected)
	}
	return nil
}

func requireText(path, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be empty", path)
	}
	return nil
}
