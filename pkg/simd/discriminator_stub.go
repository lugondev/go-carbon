//go:build !amd64
// +build !amd64

package simd

func matchBatchAVX2Asm(targets []Discriminator, candidates []Discriminator, results []int) {
	panic("SIMD not supported on this architecture")
}

func compareDiscriminatorsAVX2Asm(target Discriminator, candidates []Discriminator, results []bool) {
	panic("SIMD not supported on this architecture")
}
