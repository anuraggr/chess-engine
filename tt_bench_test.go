package main

import (
	"math/rand"
	"testing"
)

var benchKeys []uint64

func init() {
	benchKeys = make([]uint64, 1000000)
	for i := range benchKeys {
		benchKeys[i] = rand.Uint64()
	}
}

func BenchmarkTTStore(b *testing.B) {
	tt := NewTT(16) // use a 16MB table
	m := Move{From: 12, To: 28, Flag: FlagDoublePush}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := benchKeys[i%len(benchKeys)]
		tt.Store(key, uint8(i%10), int16(i%200), FlagAlpha, m)
	}
}

func BenchmarkTTProbeMiss(b *testing.B) {
	tt := NewTT(16)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := benchKeys[i%len(benchKeys)]
		tt.Probe(key)
	}
}

func BenchmarkTTProbeHit(b *testing.B) {
	tt := NewTT(16)
	m := Move{From: 12, To: 28, Flag: FlagDoublePush}
	
	// Pre-fill
	for i := 0; i < len(benchKeys); i++ {
		tt.Store(benchKeys[i], 10, 50, FlagExact, m)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := benchKeys[i%len(benchKeys)]
		tt.Probe(key)
	}
}

func BenchmarkTTRandomMix_25Store_75Probe(b *testing.B) {
	tt := NewTT(16)
	m := Move{From: 12, To: 28, Flag: FlagDoublePush}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := benchKeys[i%len(benchKeys)]
		if i%4 == 0 {
			tt.Store(key, uint8(i%10), int16(i%200), FlagExact, m)
		} else {
			tt.Probe(key)
		}
	}
}

func BenchmarkTTRandomMix_50Store_50Probe(b *testing.B) {
	tt := NewTT(16)
	m := Move{From: 12, To: 28, Flag: FlagDoublePush}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := benchKeys[i%len(benchKeys)]
		if i%2 == 0 {
			tt.Store(key, uint8(i%10), int16(i%200), FlagBeta, m)
		} else {
			tt.Probe(key)
		}
	}
}
