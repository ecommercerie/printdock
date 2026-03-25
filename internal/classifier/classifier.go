package classifier

import (
	"math"
	"printdock/internal/rules"
	"regexp"
	"sync"
)

type Classifier struct {
	mu       sync.RWMutex
	rules    []rules.Rule
	compiled []*regexp.Regexp
}

func New(rulesList []rules.Rule) *Classifier {
	c := &Classifier{}
	c.loadRules(rulesList)
	return c
}

func (c *Classifier) loadRules(rulesList []rules.Rule) {
	compiled := make([]*regexp.Regexp, len(rulesList))
	for i, r := range rulesList {
		if r.Match.FilenameRegex != "" {
			re, err := regexp.Compile(r.Match.FilenameRegex)
			if err == nil {
				compiled[i] = re
			}
		}
	}
	c.rules = rulesList
	c.compiled = compiled
}

func (c *Classifier) Reload(rulesList []rules.Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadRules(rulesList)
}

func (c *Classifier) Classify(filename string, widthMM, heightMM float64, pageCount int) *rules.Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i, rule := range c.rules {
		if !c.matches(i, rule, filename, widthMM, heightMM, pageCount) {
			continue
		}
		r := rule
		return &r
	}
	return nil
}

func (c *Classifier) matches(idx int, rule rules.Rule, filename string, widthMM, heightMM float64, pageCount int) bool {
	m := rule.Match

	if m.FilenameRegex != "" {
		if c.compiled[idx] == nil || !c.compiled[idx].MatchString(filename) {
			return false
		}
	}

	tolerance := m.DimensionTolerance
	if tolerance <= 0 {
		tolerance = 5
	}

	if m.PageWidthMM > 0 {
		if math.Abs(widthMM-m.PageWidthMM) > tolerance {
			return false
		}
	}
	if m.PageHeightMM > 0 {
		if math.Abs(heightMM-m.PageHeightMM) > tolerance {
			return false
		}
	}

	if m.PageCount > 0 && pageCount != m.PageCount {
		return false
	}

	return true
}
