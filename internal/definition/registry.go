package definition

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/pitabwire/thesa/model"
)

// snapshot is an immutable collection of all definitions indexed by ID.
type snapshot struct {
	domains   map[string]model.DomainDefinition
	pages     map[string]model.PageDefinition
	forms     map[string]model.FormDefinition
	commands  map[string]model.CommandDefinition
	workflows map[string]model.WorkflowDefinition
	searches  map[string]model.SearchDefinition
	lookups   map[string]model.LookupDefinition
	checksum  string
}

// Registry is a read-optimized, thread-safe store of all loaded definitions.
// It uses atomic pointer swap for lock-free concurrent reads.
type Registry struct {
	snap atomic.Pointer[snapshot]
}

// NewRegistry creates a Registry from the given definitions.
func NewRegistry(defs []model.DomainDefinition) *Registry {
	r := &Registry{}
	r.Replace(defs)
	return r
}

// Replace atomically swaps the registry contents with a new snapshot built
// from the given definitions.
func (r *Registry) Replace(defs []model.DomainDefinition) {
	s := &snapshot{
		domains:   make(map[string]model.DomainDefinition, len(defs)),
		pages:     make(map[string]model.PageDefinition),
		forms:     make(map[string]model.FormDefinition),
		commands:  make(map[string]model.CommandDefinition),
		workflows: make(map[string]model.WorkflowDefinition),
		searches:  make(map[string]model.SearchDefinition),
		lookups:   make(map[string]model.LookupDefinition),
	}

	var checksumParts []string

	for _, def := range defs {
		s.domains[def.Domain] = def
		checksumParts = append(checksumParts, def.Checksum)

		for _, p := range def.Pages {
			s.pages[p.ID] = p
		}
		for _, f := range def.Forms {
			s.forms[f.ID] = f
		}
		for _, c := range def.Commands {
			s.commands[c.ID] = c
		}
		for _, w := range def.Workflows {
			s.workflows[w.ID] = w
		}
		for _, sr := range def.Searches {
			s.searches[sr.ID] = sr
		}
		for _, l := range def.Lookups {
			s.lookups[l.ID] = l
		}
	}

	sort.Strings(checksumParts)
	combined := strings.Join(checksumParts, ":")
	s.checksum = fmt.Sprintf("%x", sha256.Sum256([]byte(combined)))

	r.snap.Store(s)
}

func (r *Registry) current() *snapshot {
	return r.snap.Load()
}

// GetDomain returns the domain definition with the given ID.
func (r *Registry) GetDomain(domainID string) (model.DomainDefinition, bool) {
	d, ok := r.current().domains[domainID]
	return d, ok
}

// GetPage returns the page definition with the given ID.
func (r *Registry) GetPage(pageID string) (model.PageDefinition, bool) {
	p, ok := r.current().pages[pageID]
	return p, ok
}

// GetForm returns the form definition with the given ID.
func (r *Registry) GetForm(formID string) (model.FormDefinition, bool) {
	f, ok := r.current().forms[formID]
	return f, ok
}

// GetCommand returns the command definition with the given ID.
func (r *Registry) GetCommand(commandID string) (model.CommandDefinition, bool) {
	c, ok := r.current().commands[commandID]
	return c, ok
}

// GetWorkflow returns the workflow definition with the given ID.
func (r *Registry) GetWorkflow(workflowID string) (model.WorkflowDefinition, bool) {
	w, ok := r.current().workflows[workflowID]
	return w, ok
}

// GetSearch returns the search definition with the given ID.
func (r *Registry) GetSearch(searchID string) (model.SearchDefinition, bool) {
	s, ok := r.current().searches[searchID]
	return s, ok
}

// GetLookup returns the lookup definition with the given ID.
func (r *Registry) GetLookup(lookupID string) (model.LookupDefinition, bool) {
	l, ok := r.current().lookups[lookupID]
	return l, ok
}

// AllDomains returns all domain definitions.
func (r *Registry) AllDomains() []model.DomainDefinition {
	s := r.current()
	defs := make([]model.DomainDefinition, 0, len(s.domains))
	for _, d := range s.domains {
		defs = append(defs, d)
	}
	return defs
}

// AllSearches returns all search definitions.
func (r *Registry) AllSearches() []model.SearchDefinition {
	s := r.current()
	defs := make([]model.SearchDefinition, 0, len(s.searches))
	for _, sr := range s.searches {
		defs = append(defs, sr)
	}
	return defs
}

// Checksum returns the combined checksum of all loaded definitions.
func (r *Registry) Checksum() string {
	return r.current().checksum
}
