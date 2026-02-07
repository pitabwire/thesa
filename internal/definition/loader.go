// Package definition loads YAML definitions, validates them against OpenAPI specs,
// and provides a fast-lookup registry with atomic pointer swap.
package definition

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pitabwire/thesa/model"
	"gopkg.in/yaml.v3"
)

// Loader scans directories for YAML definition files, parses them, and computes
// SHA-256 checksums.
type Loader struct{}

// NewLoader creates a new definition Loader.
func NewLoader() *Loader {
	return &Loader{}
}

// LoadAll recursively scans directories for *.yaml and *.yml files and parses
// each into a DomainDefinition.
func (l *Loader) LoadAll(directories []string) ([]model.DomainDefinition, error) {
	var defs []model.DomainDefinition

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}

			def, err := l.LoadFile(path)
			if err != nil {
				return fmt.Errorf("loading %s: %w", path, err)
			}
			defs = append(defs, def)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scanning directory %s: %w", dir, err)
		}
	}

	return defs, nil
}

// LoadFile loads and parses a single YAML definition file. It computes the
// SHA-256 checksum and records the source file path.
func (l *Loader) LoadFile(path string) (model.DomainDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.DomainDefinition{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var def model.DomainDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return model.DomainDefinition{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	def.Checksum = checksum
	def.SourceFile = path

	return def, nil
}
