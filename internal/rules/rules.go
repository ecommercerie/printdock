package rules

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Rule represents a single routing rule mapping file characteristics to a printer action.
type Rule struct {
	Name   string     `yaml:"name" json:"name"`
	Match  RuleMatch  `yaml:"match" json:"match"`
	Action RuleAction `yaml:"action" json:"action"`
}

// RuleMatch holds the criteria used to match an incoming document.
type RuleMatch struct {
	FilenameRegex      string  `yaml:"filename_regex,omitempty" json:"filenameRegex"`
	PageWidthMM        float64 `yaml:"page_width_mm,omitempty" json:"pageWidthMm"`
	PageHeightMM       float64 `yaml:"page_height_mm,omitempty" json:"pageHeightMm"`
	DimensionTolerance float64 `yaml:"dimension_tolerance_mm,omitempty" json:"dimensionToleranceMm"`
	PageCount          int     `yaml:"page_count,omitempty" json:"pageCount"`
}

// RuleAction describes what to do when a rule matches.
type RuleAction struct {
	Printer       string `yaml:"printer" json:"printer"`
	ArchiveSubdir string `yaml:"archive_subdir" json:"archiveSubdir"`
}

// ruleFile is used to support YAML files with a top-level "rules:" key.
type ruleFile struct {
	Rules []Rule `yaml:"rules"`
}

// Load is a package-level function that reads rules from a YAML file at the given path.
// If the file does not exist it returns an empty slice without error.
// Supports both a bare list format and a top-level "rules:" key format.
func Load(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Rule{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("rules: read %s: %w", path, err)
	}

	if len(data) == 0 {
		return []Rule{}, nil
	}

	// Try the "rules:" wrapper format first.
	var wrapped ruleFile
	if err := yaml.Unmarshal(data, &wrapped); err == nil && wrapped.Rules != nil {
		return wrapped.Rules, nil
	}

	var out []Rule
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("rules: parse %s: %w", path, err)
	}
	if out == nil {
		out = []Rule{}
	}
	return out, nil
}

// save is a package-level helper that persists rules to disk as a bare YAML list.
func save(path string, rules []Rule) error {
	data, err := yaml.Marshal(rules)
	if err != nil {
		return fmt.Errorf("rules: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("rules: write %s: %w", path, err)
	}
	return nil
}

// Store provides thread-safe CRUD access to a rules YAML file.
type Store struct {
	mu   sync.Mutex
	path string
}

// NewStore creates a new Store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load reads and returns the current rules from disk.
func (s *Store) Load() ([]Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Load(s.path)
}

// Save persists the given rules to disk, replacing any existing content.
func (s *Store) Save(rules []Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(rules)
}

// save is the internal (non-locking) save helper.
func (s *Store) save(rules []Rule) error {
	return save(s.path, rules)
}

// Add appends a new rule. Returns an error if a rule with the same name already exists.
func (s *Store) Add(rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := Load(s.path)
	if err != nil {
		return err
	}

	for _, r := range existing {
		if r.Name == rule.Name {
			return fmt.Errorf("rules: rule %q already exists", rule.Name)
		}
	}

	return s.save(append(existing, rule))
}

// Delete removes the rule with the given name. Returns an error if it does not exist.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := Load(s.path)
	if err != nil {
		return err
	}

	filtered := existing[:0:0] // preserve nil vs empty semantics safely
	found := false
	for _, r := range existing {
		if r.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("rules: rule %q not found", name)
	}

	return s.save(filtered)
}

// Reorder sets the rules list to the given order (by names).
// All names must exist. Returns error if a name is not found.
func (s *Store) Reorder(names []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, err := Load(s.path)
	if err != nil {
		return err
	}

	byName := make(map[string]Rule, len(existing))
	for _, r := range existing {
		byName[r.Name] = r
	}

	if len(names) != len(existing) {
		return fmt.Errorf("reorder: expected %d names, got %d", len(existing), len(names))
	}

	reordered := make([]Rule, 0, len(names))
	for _, name := range names {
		r, ok := byName[name]
		if !ok {
			return fmt.Errorf("reorder: rule %q not found", name)
		}
		reordered = append(reordered, r)
	}
	return save(s.path, reordered)
}

// Update replaces a rule by name with the new rule data.
func (s *Store) Update(name string, updated Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, err := Load(s.path)
	if err != nil {
		return err
	}
	found := false
	for i, r := range existing {
		if r.Name == name {
			existing[i] = updated
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("update: rule %q not found", name)
	}
	return save(s.path, existing)
}
