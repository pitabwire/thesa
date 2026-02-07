package capability

import (
	"fmt"
	"os"
	"sync"

	"github.com/pitabwire/thesa/model"
	"gopkg.in/yaml.v3"
)

type policyFile struct {
	Roles map[string][]string `yaml:"roles"`
}

// StaticPolicyEvaluator resolves capabilities from a static YAML file
// mapping roles to capability strings.
type StaticPolicyEvaluator struct {
	path   string
	mu     sync.RWMutex
	policy policyFile
}

// NewStaticPolicyEvaluator creates a new evaluator that loads policies from path.
func NewStaticPolicyEvaluator(path string) (*StaticPolicyEvaluator, error) {
	e := &StaticPolicyEvaluator{path: path}
	if err := e.Sync(); err != nil {
		return nil, err
	}
	return e, nil
}

// ResolveCapabilities returns the union of capabilities for all roles in the
// request context.
func (e *StaticPolicyEvaluator) ResolveCapabilities(rctx *model.RequestContext) (model.CapabilitySet, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	caps := make(model.CapabilitySet)
	for _, role := range rctx.Roles {
		for _, cap := range e.policy.Roles[role] {
			caps[cap] = true
		}
	}
	return caps, nil
}

// Evaluate checks a single capability against the resolved set.
func (e *StaticPolicyEvaluator) Evaluate(rctx *model.RequestContext, capability string, _ map[string]any) (bool, error) {
	caps, err := e.ResolveCapabilities(rctx)
	if err != nil {
		return false, err
	}
	return caps.Has(capability), nil
}

// EvaluateAll checks multiple capabilities at once.
func (e *StaticPolicyEvaluator) EvaluateAll(rctx *model.RequestContext, capabilities []string, _ map[string]any) (map[string]bool, error) {
	caps, err := e.ResolveCapabilities(rctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(capabilities))
	for _, cap := range capabilities {
		result[cap] = caps.Has(cap)
	}
	return result, nil
}

// Sync reloads the policy file from disk.
func (e *StaticPolicyEvaluator) Sync() error {
	data, err := os.ReadFile(e.path)
	if err != nil {
		return fmt.Errorf("capability: reading policy file %s: %w", e.path, err)
	}

	var p policyFile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("capability: parsing policy file %s: %w", e.path, err)
	}

	e.mu.Lock()
	e.policy = p
	e.mu.Unlock()

	return nil
}
