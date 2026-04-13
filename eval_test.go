package main

import "testing"

func TestEvalStartingPosition(t *testing.T) {
	b := NewBoard()
	score := Evaluate(b)

	// Starting position is symmetric, score should be exactly 0
	if score != 0 {
		t.Errorf("Starting position eval = %d, want 0", score)
	} else {
		t.Logf("Starting position eval = %d ✓", score)
	}
}

func TestEvalWhiteQueenAdvantage(t *testing.T) {
	// White has an extra queen (black is missing their queen)
	b, err := BoardFromFEN("rnb1kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	score := Evaluate(b)
	// White should be significantly ahead (at least 800 cp with the queen)
	if score < 800 {
		t.Errorf("White +Q eval = %d, want >= 800", score)
	} else {
		t.Logf("White +Q eval = %d ✓", score)
	}
}

func TestEvalBlackRookAdvantage(t *testing.T) {
	// Black has an extra rook
	b, err := BoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/1NBQKBNr w Qkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	score := Evaluate(b)
	// Black should be ahead (so negative score)
	if score >= 0 {
		t.Errorf("Black +R eval = %d, want < 0", score)
	} else {
		t.Logf("Black +R eval = %d ✓", score)
	}
}

func TestEvalMaterialOnly(t *testing.T) {
	// Kings only position should be roughly equal (around zero).
	// but because of PST and mobility, one side might have a small advantage
	b, err := BoardFromFEN("8/8/8/8/8/8/8/4K2k w - - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	score := Evaluate(b)
	if score > 50 || score < -50 {
		t.Errorf("Kings-only eval = %d, want near 0", score)
	} else {
		t.Logf("Kings-only eval = %d ✓", score)
	}
}
