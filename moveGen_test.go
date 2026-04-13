package main

import (
	"testing"
)

// perft counts the number of leaf nodes at a given depth.
func perft(b *Board, depth int) int64 {
	if depth == 0 {
		return 1
	}

	moves := GenerateLegalMoves(b)
	if depth == 1 {
		return int64(len(moves))
	}

	var nodes int64
	for _, m := range moves {
		nb := MakeMove(b, m)
		nodes += perft(nb, depth-1)
	}
	return nodes
}

// prrft tests — starting position
func TestPerftStartingPosition(t *testing.T) {
	b := NewBoard()

	tests := []struct {
		depth    int
		expected int64
	}{
		{1, 20},
		{2, 400},
		{3, 8902},
		{4, 197281},
	}

	for _, tt := range tests {
		got := perft(b, tt.depth)
		if got != tt.expected {
			t.Errorf("Perft(%d) starting position = %d, want %d", tt.depth, got, tt.expected)
		} else {
			t.Logf("Perft(%d) starting position = %d ✓", tt.depth, got)
		}
	}
}

// Perft tests — Kiwipete position -> very famous for perft tests
func TestPerftKiwipete(t *testing.T) {
	b, err := BoardFromFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq -")
	if err != nil {
		t.Fatalf("Failed to parse Kiwipete FEN: %v", err)
	}

	tests := []struct {
		depth    int
		expected int64
	}{
		{1, 48},
		{2, 2039},
		{3, 97862},
	}

	for _, tt := range tests {
		got := perft(b, tt.depth)
		if got != tt.expected {
			t.Errorf("Perft(%d) Kiwipete = %d, want %d", tt.depth, got, tt.expected)
		} else {
			t.Logf("Perft(%d) Kiwipete = %d ✓", tt.depth, got)
		}
	}
}

// Perft test — Position 3 (from CPW)
func TestPerftPosition3(t *testing.T) {
	b, err := BoardFromFEN("8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - -")
	if err != nil {
		t.Fatalf("Failed to parse position 3 FEN: %v", err)
	}

	tests := []struct {
		depth    int
		expected int64
	}{
		{1, 14},
		{2, 191},
		{3, 2812},
		{4, 43238},
	}

	for _, tt := range tests {
		got := perft(b, tt.depth)
		if got != tt.expected {
			t.Errorf("Perft(%d) Position3 = %d, want %d", tt.depth, got, tt.expected)
		} else {
			t.Logf("Perft(%d) Position3 = %d ✓", tt.depth, got)
		}
	}
}

// FEN round-trip

func TestFENRoundTrip(t *testing.T) {
	fens := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
	}

	for _, fen := range fens {
		b, err := BoardFromFEN(fen)
		if err != nil {
			t.Errorf("Failed to parse FEN %q: %v", fen, err)
			continue
		}
		got := b.FEN()
		if got != fen {
			t.Errorf("FEN round-trip failed:\n  input:  %s\n  output: %s", fen, got)
		}
	}
}

// Check detection

func TestIsInCheck(t *testing.T) {
	// Scholar's mate position — black king in check
	b, err := BoardFromFEN("r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}
	if !IsInCheck(b, Black) {
		t.Error("Expected Black to be in check")
	}
	if IsInCheck(b, White) {
		t.Error("Expected White to NOT be in check")
	}
}

// Castling

func TestCastling(t *testing.T) {
	// Position where both sides can castle both ways
	b, err := BoardFromFEN("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	moves := GenerateLegalMoves(b)
	var hasKingside, hasQueenside bool
	for _, m := range moves {
		if m.Flag == FlagCastleKing {
			hasKingside = true
		}
		if m.Flag == FlagCastleQueen {
			hasQueenside = true
		}
	}

	if !hasKingside {
		t.Error("Expected kingside castling to be available")
	}
	if !hasQueenside {
		t.Error("Expected queenside castling to be available")
	}
}

// Promotion
func TestPromotion(t *testing.T) {
	// white pawn on e7, should promote
	b, err := BoardFromFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	moves := GenerateLegalMoves(b)
	promoCount := 0
	for _, m := range moves {
		if m.Flag == FlagPromotion {
			promoCount++
		}
	}

	if promoCount != 4 {
		t.Errorf("Expected 4 promotion moves, got %d", promoCount)
	}
}

// En passant
func TestEnPassant(t *testing.T) {
	// white pawn on e5, black just played d7-d5
	b, err := BoardFromFEN("rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	moves := GenerateLegalMoves(b)
	hasEP := false
	for _, m := range moves {
		if m.Flag == FlagEnPassant {
			hasEP = true
			break
		}
	}

	if !hasEP {
		t.Error("Expected en passant move to be available")
	}
}

// Checkmate detection
func TestCheckmate(t *testing.T) {
	// Fool's mate position
	b, err := BoardFromFEN("rnb1kbnr/pppp1ppp/4p3/8/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	outcome, method := GetOutcome(b)
	if outcome != BlackWins {
		t.Errorf("Expected BlackWins, got %s", outcome)
	}
	if method != Checkmate {
		t.Errorf("Expected Checkmate, got %s", method)
	}
}

// Stalemate detection
func TestStalemate(t *testing.T) {
	// Black king on a8, white king on a6, white queen on b6 — stalemate
	b, err := BoardFromFEN("k7/8/KQ6/8/8/8/8/8 b - - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	outcome, method := GetOutcome(b)
	if outcome != Draw {
		t.Errorf("Expected Draw, got %s", outcome)
	}
	if method != Stalemate {
		t.Errorf("Expected Stalemate, got %s", method)
	}
}
