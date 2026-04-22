package main

import (
	"testing"
)

const (
	benchPosStart      = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	benchPosMiddlegame = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	benchPosEndgame    = "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1"
)

func benchmarkBoards(b *testing.B) []*Board {
	b1, err1 := BoardFromFEN(benchPosStart)
	if err1 != nil {
		b.Fatalf("Failed to parse start FEN: %v", err1)
	}
	b2, err2 := BoardFromFEN(benchPosMiddlegame)
	if err2 != nil {
		b.Fatalf("Failed to parse middlegame FEN: %v", err2)
	}
	b3, err3 := BoardFromFEN(benchPosEndgame)
	if err3 != nil {
		b.Fatalf("Failed to parse endgame FEN: %v", err3)
	}
	return []*Board{b1, b2, b3}
}

func BenchmarkEvaluate(b *testing.B) {
	boards := benchmarkBoards(b)
	names := []string{"Startpos", "Middlegame", "Endgame"}
	for idx, board := range boards {
		b.Run(names[idx], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Evaluate(board)
			}
		})
	}
}

func BenchmarkEvalMaterial(b *testing.B) {
	boards := benchmarkBoards(b)
	names := []string{"Startpos", "Middlegame", "Endgame"}
	for idx, board := range boards {
		b.Run(names[idx], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				evalMaterial(board)
			}
		})
	}
}

func BenchmarkEvalPST(b *testing.B) {
	boards := benchmarkBoards(b)
	names := []string{"Startpos", "Middlegame", "Endgame"}
	for idx, board := range boards {
		b.Run(names[idx], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				evalPST(board)
			}
		})
	}
}

func BenchmarkEvalMobility(b *testing.B) {
	boards := benchmarkBoards(b)
	names := []string{"Startpos", "Middlegame", "Endgame"}
	for idx, board := range boards {
		b.Run(names[idx], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				evalMobility(board)
			}
		})
	}
}

func BenchmarkEvalPawnStructure(b *testing.B) {
	boards := benchmarkBoards(b)
	names := []string{"Startpos", "Middlegame", "Endgame"}
	for idx, board := range boards {
		b.Run(names[idx], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				evalPawnStructure(board)
			}
		})
	}
}

func BenchmarkEvalKingSafety(b *testing.B) {
	boards := benchmarkBoards(b)
	names := []string{"Startpos", "Middlegame", "Endgame"}
	for idx, board := range boards {
		b.Run(names[idx], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				evalKingSafety(board)
			}
		})
	}
}
