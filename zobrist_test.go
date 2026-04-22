package main

import (
	"testing"
)

// TestPolyglotStartingPosition verifies that the Zobrist hash of the standard
// starting position matches the well-known Polyglot value.
func TestPolyglotStartingPosition(t *testing.T) {
	b := NewBoard()
	const expected uint64 = 0x463b96181691fc9c
	if b.Hash != expected {
		t.Errorf("Starting position hash = 0x%016x, want 0x%016x", b.Hash, expected)
	}
}

// TestZobristConsistency computes the hash from scratch and verifies it
// matches b.Hash for several complex FEN positions.
func TestZobristConsistency(t *testing.T) {
	fens := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/3P1N1P/PPP1NPP1/R2Q1RK1 w - - 0 1",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
	}

	for _, fen := range fens {
		b, err := BoardFromFEN(fen)
		if err != nil {
			t.Fatalf("Failed to parse FEN %q: %v", fen, err)
		}

		hashFromInit := b.Hash

		// Recompute the hash from scratch.
		b.ComputeHash()

		if b.Hash != hashFromInit {
			t.Errorf("FEN %q: ComputeHash() = 0x%016x, but initial Hash = 0x%016x",
				fen, b.Hash, hashFromInit)
		}
	}
}

// TestZobristIncrementalMatchesFull makes a sequence of moves from the
// starting position and verifies that the incrementally-updated b.Hash
// equals a full recomputation after each move.
func TestZobristIncrementalMatchesFull(t *testing.T) {
	b := NewBoard()

	// A short opening sequence: 1. e4 e5 2. Nf3 Nc6 3. Bb5 (Ruy Lopez)
	moves := []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1b5"}

	for _, ms := range moves {
		from, err := SquareFromAlgebraic(ms[:2])
		if err != nil {
			t.Fatalf("bad from square %q: %v", ms[:2], err)
		}
		to, err := SquareFromAlgebraic(ms[2:4])
		if err != nil {
			t.Fatalf("bad to square %q: %v", ms[2:4], err)
		}

		// Find the legal move matching from/to.
		legal := GenerateLegalMoves(b)
		var found bool
		for _, m := range legal {
			if m.From == from && m.To == to {
				b = MakeMove(b, m)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("move %s not found in legal moves", ms)
		}

		// Recompute the hash from scratch and compare.
		incremental := b.Hash
		b.ComputeHash()
		if b.Hash != incremental {
			t.Errorf("After %s: incremental hash 0x%016x != recomputed 0x%016x",
				ms, incremental, b.Hash)
		}
	}
}

// TestZobristDifferentPositions verifies that different positions produce
// different Zobrist hashes.
func TestZobristDifferentPositions(t *testing.T) {
	fens := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"rnbqkbnr/pppp1ppp/4p3/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	}

	hashes := make(map[uint64]string)
	for _, fen := range fens {
		b, err := BoardFromFEN(fen)
		if err != nil {
			t.Fatalf("Failed to parse FEN %q: %v", fen, err)
		}
		if prev, exists := hashes[b.Hash]; exists {
			t.Errorf("Hash collision: FEN %q has same hash as %q (0x%016x)",
				fen, prev, b.Hash)
		}
		hashes[b.Hash] = fen
	}
}

// TestThreefoldRepetition plays Ng1-f3, Nb8-c6, Nf3-g1, Nc6-b8 twice
// to reach the starting position 3 times total, then verifies that
// GetOutcome returns Draw via ThreefoldRepetition.
func TestThreefoldRepetition(t *testing.T) {
	b := NewBoard()

	// Each cycle: Nf3 Nc6 Ng1 Nb8 returns to the start position.
	cycle := [][2]string{
		{"g1", "f3"}, // Ng1-f3
		{"b8", "c6"}, // Nb8-c6
		{"f3", "g1"}, // Nf3-g1
		{"c6", "b8"}, // Nc6-b8
	}

	for rep := 0; rep < 2; rep++ {
		for _, mv := range cycle {
			from, _ := SquareFromAlgebraic(mv[0])
			to, _ := SquareFromAlgebraic(mv[1])

			legal := GenerateLegalMoves(b)
			var found bool
			for _, m := range legal {
				if m.From == from && m.To == to {
					b = MakeMove(b, m)
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Cycle %d: move %s%s not found in legal moves", rep, mv[0], mv[1])
			}
		}
	}

	outcome, method := GetOutcome(b)
	if outcome != Draw {
		t.Errorf("Expected Draw, got %s", outcome)
	}
	if method != ThreefoldRepetition {
		t.Errorf("Expected ThreefoldRepetition, got %s", method)
	}
}

// TestNoFalseRepetition plays a few non-repeating moves and confirms no
// repetition draw is triggered.
func TestNoFalseRepetition(t *testing.T) {
	b := NewBoard()

	// Play 1. e4 e5 2. Nf3 Nc6 — no repeated positions.
	moves := []string{"e2e4", "e7e5", "g1f3", "b8c6"}

	for _, ms := range moves {
		from, err := SquareFromAlgebraic(ms[:2])
		if err != nil {
			t.Fatalf("bad from square %q: %v", ms[:2], err)
		}
		to, err := SquareFromAlgebraic(ms[2:4])
		if err != nil {
			t.Fatalf("bad to square %q: %v", ms[2:4], err)
		}

		legal := GenerateLegalMoves(b)
		var found bool
		for _, m := range legal {
			if m.From == from && m.To == to {
				b = MakeMove(b, m)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("move %s not found in legal moves", ms)
		}
	}

	outcome, method := GetOutcome(b)
	if outcome != NoOutcome {
		t.Errorf("Expected NoOutcome, got %s (method: %s)", outcome, method)
	}
}
