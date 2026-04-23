package main

import "testing"

func TestEvalStartingPosition(t *testing.T) {
	b := NewBoard()
	score := Evaluate(b)

	if score != 0 {
		t.Errorf("Starting position eval = %d, want 0", score)
	} else {
		t.Logf("Starting position eval = %d ✓", score)
	}
}

func TestEvalWhiteQueenAdvantage(t *testing.T) {
	b, err := BoardFromFEN("rnb1kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	score := Evaluate(b)
	if score < 800 {
		t.Errorf("White +Q eval = %d, want >= 800", score)
	} else {
		t.Logf("White +Q eval = %d ✓", score)
	}
}

func TestEvalBlackRookAdvantage(t *testing.T) {
	b, err := BoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/1NBQKBNr w Qkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	score := Evaluate(b)
	if score >= 0 {
		t.Errorf("Black +R eval = %d, want < 0", score)
	} else {
		t.Logf("Black +R eval = %d ✓", score)
	}
}

func TestEvalMaterialOnly(t *testing.T) {
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

func TestEvalPassedPawn(t *testing.T) {
	bPassed, err := BoardFromFEN("k7/p7/8/7P/8/8/8/K7 w - - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}
	scorePassed := Evaluate(bPassed)

	bBlocked, err := BoardFromFEN("k7/7p/8/7P/8/8/8/K7 w - - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}
	scoreBlocked := Evaluate(bBlocked)

	if scorePassed <= scoreBlocked {
		t.Errorf("Expected pawn on h5 to evaluate correctly higher when passed. scorePassed: %d, scoreBlocked: %d", scorePassed, scoreBlocked)
	} else {
		t.Logf("Passed pawn evaluated higher than blocked! scorePassed: %d, scoreBlocked: %d ✓", scorePassed, scoreBlocked)
	}
}
