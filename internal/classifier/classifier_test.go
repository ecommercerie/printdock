package classifier

import (
	"printdock/internal/rules"
	"testing"
)

func TestMatchByFilename(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "label", Match: rules.RuleMatch{FilenameRegex: "(?i)label"}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	result := c.Classify("order-label-123.pdf", 100, 150, 1)
	if result == nil || result.Name != "label" {
		t.Errorf("expected match on label rule")
	}
}

func TestMatchByDimensions(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "a4", Match: rules.RuleMatch{PageWidthMM: 210, PageHeightMM: 297, DimensionTolerance: 5}, Action: rules.RuleAction{Printer: "HP"}},
	})
	result := c.Classify("random.pdf", 211, 296, 1)
	if result == nil || result.Name != "a4" {
		t.Errorf("expected match on a4 rule (within tolerance)")
	}
}

func TestNoMatch(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "label", Match: rules.RuleMatch{FilenameRegex: "(?i)label"}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	result := c.Classify("invoice.pdf", 210, 297, 1)
	if result != nil {
		t.Errorf("expected no match, got %s", result.Name)
	}
}

func TestDimensionOutOfTolerance(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "thermal", Match: rules.RuleMatch{PageWidthMM: 100, PageHeightMM: 150, DimensionTolerance: 5}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	result := c.Classify("file.pdf", 120, 150, 1)
	if result != nil {
		t.Errorf("expected no match (width 120 too far from 100)")
	}
}

func TestMatchByPageCount(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "multi", Match: rules.RuleMatch{PageCount: 3}, Action: rules.RuleAction{Printer: "HP"}},
	})
	if result := c.Classify("doc.pdf", 210, 297, 3); result == nil {
		t.Errorf("expected match on page count")
	}
	if result := c.Classify("doc.pdf", 210, 297, 1); result != nil {
		t.Errorf("expected no match (wrong page count)")
	}
}

func TestANDLogic(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "strict", Match: rules.RuleMatch{FilenameRegex: "(?i)label", PageWidthMM: 100, PageHeightMM: 150, DimensionTolerance: 5}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	if result := c.Classify("label.pdf", 210, 297, 1); result != nil {
		t.Errorf("expected no match (dimensions mismatch)")
	}
	if result := c.Classify("label.pdf", 100, 150, 1); result == nil {
		t.Errorf("expected match")
	}
}

func TestFirstMatchWins(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "first", Match: rules.RuleMatch{FilenameRegex: "(?i)doc"}, Action: rules.RuleAction{Printer: "P1"}},
		{Name: "second", Match: rules.RuleMatch{FilenameRegex: "(?i)doc"}, Action: rules.RuleAction{Printer: "P2"}},
	})
	result := c.Classify("doc.pdf", 0, 0, 1)
	if result == nil || result.Name != "first" {
		t.Errorf("expected first rule to win")
	}
}

func TestReload(t *testing.T) {
	c := New([]rules.Rule{})
	if result := c.Classify("label.pdf", 0, 0, 1); result != nil {
		t.Errorf("expected no match with empty rules")
	}
	c.Reload([]rules.Rule{
		{Name: "label", Match: rules.RuleMatch{FilenameRegex: "(?i)label"}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	if result := c.Classify("label.pdf", 0, 0, 1); result == nil {
		t.Errorf("expected match after reload")
	}
}
