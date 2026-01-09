package simd

import (
	"runtime"
)

type Discriminator [8]byte

type MatcherStrategy int

const (
	StrategyAuto MatcherStrategy = iota
	StrategyMap
	StrategySIMD
	StrategyHybrid
)

type DiscriminatorMatcher struct {
	discriminators map[Discriminator]int
	orderedDiscs   []Discriminator
	strategy       MatcherStrategy
	simdEnabled    bool
}

func NewDiscriminatorMatcher(discs []Discriminator, strategy MatcherStrategy) *DiscriminatorMatcher {
	m := &DiscriminatorMatcher{
		discriminators: make(map[Discriminator]int, len(discs)),
		orderedDiscs:   make([]Discriminator, len(discs)),
		strategy:       strategy,
		simdEnabled:    runtime.GOARCH == "amd64",
	}

	for i, disc := range discs {
		m.discriminators[disc] = i
		m.orderedDiscs[i] = disc
	}

	return m
}

func (m *DiscriminatorMatcher) Match(target Discriminator) int {
	if idx, exists := m.discriminators[target]; exists {
		return idx
	}
	return -1
}

func (m *DiscriminatorMatcher) MatchBatch(targets []Discriminator) []int {
	if len(targets) == 0 {
		return nil
	}

	strategy := m.selectStrategy(len(targets))

	switch strategy {
	case StrategyMap:
		return m.matchBatchMap(targets)
	case StrategySIMD:
		return m.matchBatchSIMD(targets)
	case StrategyHybrid:
		return m.matchBatchHybrid(targets)
	default:
		return m.matchBatchMap(targets)
	}
}

func (m *DiscriminatorMatcher) selectStrategy(batchSize int) MatcherStrategy {
	if m.strategy != StrategyAuto {
		return m.strategy
	}

	const (
		mapThreshold    = 10
		simdThreshold   = 100
		hybridThreshold = 50
	)

	if batchSize < mapThreshold {
		return StrategyMap
	}

	if batchSize >= simdThreshold && m.simdEnabled {
		return StrategySIMD
	}

	if batchSize >= hybridThreshold && m.simdEnabled {
		return StrategyHybrid
	}

	return StrategyMap
}

func (m *DiscriminatorMatcher) matchBatchMap(targets []Discriminator) []int {
	results := make([]int, len(targets))
	for i := range results {
		results[i] = -1
	}

	for i, target := range targets {
		if idx, exists := m.discriminators[target]; exists {
			results[i] = idx
		}
	}

	return results
}

func (m *DiscriminatorMatcher) matchBatchSIMD(targets []Discriminator) []int {
	results := make([]int, len(targets))
	for i := range results {
		results[i] = -1
	}

	for i, target := range targets {
		for j, candidate := range m.orderedDiscs {
			if target == candidate {
				results[i] = j
				break
			}
		}
	}

	return results
}

func (m *DiscriminatorMatcher) matchBatchHybrid(targets []Discriminator) []int {
	return m.matchBatchMap(targets)
}

func CompareDiscriminators(a, b Discriminator) bool {
	return a == b
}

func CompareDiscriminatorsBatch(target Discriminator, candidates []Discriminator) []bool {
	results := make([]bool, len(candidates))

	for i, candidate := range candidates {
		results[i] = (target == candidate)
	}

	return results
}
