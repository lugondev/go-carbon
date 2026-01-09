package simd

import (
	"testing"
)

func TestNewDiscriminatorMatcher(t *testing.T) {
	discs := []Discriminator{
		{1, 2, 3, 4, 5, 6, 7, 8},
		{8, 7, 6, 5, 4, 3, 2, 1},
		{0, 0, 0, 0, 0, 0, 0, 1},
	}

	matcher := NewDiscriminatorMatcher(discs, StrategyAuto)

	if len(matcher.discriminators) != 3 {
		t.Errorf("expected 3 discriminators, got %d", len(matcher.discriminators))
	}

	if len(matcher.orderedDiscs) != 3 {
		t.Errorf("expected 3 ordered discriminators, got %d", len(matcher.orderedDiscs))
	}
}

func TestMatch(t *testing.T) {
	discs := []Discriminator{
		{1, 2, 3, 4, 5, 6, 7, 8},
		{8, 7, 6, 5, 4, 3, 2, 1},
		{0, 0, 0, 0, 0, 0, 0, 1},
	}

	matcher := NewDiscriminatorMatcher(discs, StrategyAuto)

	tests := []struct {
		name     string
		target   Discriminator
		expected int
	}{
		{
			name:     "first discriminator",
			target:   Discriminator{1, 2, 3, 4, 5, 6, 7, 8},
			expected: 0,
		},
		{
			name:     "second discriminator",
			target:   Discriminator{8, 7, 6, 5, 4, 3, 2, 1},
			expected: 1,
		},
		{
			name:     "third discriminator",
			target:   Discriminator{0, 0, 0, 0, 0, 0, 0, 1},
			expected: 2,
		},
		{
			name:     "not found",
			target:   Discriminator{9, 9, 9, 9, 9, 9, 9, 9},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.target)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestMatchBatch(t *testing.T) {
	discs := []Discriminator{
		{1, 2, 3, 4, 5, 6, 7, 8},
		{8, 7, 6, 5, 4, 3, 2, 1},
		{0, 0, 0, 0, 0, 0, 0, 1},
	}

	strategies := []struct {
		name     string
		strategy MatcherStrategy
	}{
		{"StrategyMap", StrategyMap},
		{"StrategySIMD", StrategySIMD},
		{"StrategyAuto", StrategyAuto},
	}

	for _, strat := range strategies {
		t.Run(strat.name, func(t *testing.T) {
			matcher := NewDiscriminatorMatcher(discs, strat.strategy)

			targets := []Discriminator{
				{1, 2, 3, 4, 5, 6, 7, 8},
				{9, 9, 9, 9, 9, 9, 9, 9},
				{0, 0, 0, 0, 0, 0, 0, 1},
			}

			expected := []int{0, -1, 2}

			results := matcher.MatchBatch(targets)

			if len(results) != len(expected) {
				t.Fatalf("expected %d results, got %d", len(expected), len(results))
			}

			for i, exp := range expected {
				if results[i] != exp {
					t.Errorf("result[%d]: expected %d, got %d", i, exp, results[i])
				}
			}
		})
	}
}

func TestMatchBatchEmpty(t *testing.T) {
	discs := []Discriminator{
		{1, 2, 3, 4, 5, 6, 7, 8},
	}

	matcher := NewDiscriminatorMatcher(discs, StrategyAuto)
	results := matcher.MatchBatch(nil)

	if results != nil {
		t.Errorf("expected nil for empty batch, got %v", results)
	}
}

func TestCompareDiscriminators(t *testing.T) {
	a := Discriminator{1, 2, 3, 4, 5, 6, 7, 8}
	b := Discriminator{1, 2, 3, 4, 5, 6, 7, 8}
	c := Discriminator{1, 2, 3, 4, 5, 6, 7, 9}

	if !CompareDiscriminators(a, b) {
		t.Error("expected a == b")
	}

	if CompareDiscriminators(a, c) {
		t.Error("expected a != c")
	}
}

func TestCompareDiscriminatorsBatch(t *testing.T) {
	target := Discriminator{1, 2, 3, 4, 5, 6, 7, 8}
	candidates := []Discriminator{
		{1, 2, 3, 4, 5, 6, 7, 8},
		{9, 9, 9, 9, 9, 9, 9, 9},
		{1, 2, 3, 4, 5, 6, 7, 8},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}

	expected := []bool{true, false, true, false}
	results := CompareDiscriminatorsBatch(target, candidates)

	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}

	for i, exp := range expected {
		if results[i] != exp {
			t.Errorf("result[%d]: expected %v, got %v", i, exp, results[i])
		}
	}
}

func TestSelectStrategy(t *testing.T) {
	discs := generateTestDiscriminators(10)
	matcher := NewDiscriminatorMatcher(discs, StrategyAuto)

	tests := []struct {
		batchSize int
		expected  MatcherStrategy
	}{
		{5, StrategyMap},
		{10, StrategyMap},
		{50, StrategyMap},
		{100, StrategyMap},
		{500, StrategyMap},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.batchSize)), func(t *testing.T) {
			strategy := matcher.selectStrategy(tt.batchSize)
			if strategy != tt.expected {
				t.Errorf("batch size %d: expected strategy %d, got %d",
					tt.batchSize, tt.expected, strategy)
			}
		})
	}
}
