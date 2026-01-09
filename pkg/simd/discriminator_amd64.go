//go:build amd64
// +build amd64

package simd

// matchBatchAVX2Asm uses AVX2 instructions to match multiple discriminators.
// This is implemented in discriminator_amd64.s
//
//go:noescape
func matchBatchAVX2Asm(targets []Discriminator, candidates []Discriminator, results []int)

// compareDiscriminatorsAVX2Asm uses AVX2 to compare a target against multiple candidates.
// This is implemented in discriminator_amd64.s
//
//go:noescape
func compareDiscriminatorsAVX2Asm(target Discriminator, candidates []Discriminator, results []bool)
