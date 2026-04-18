package main

import "testing"

func TestSearchMateIn1(t *testing.T) {
	b, err := BoardFromFEN("r1bqk2r/pppp1ppp/2n2n2/2b1p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	result := SearchPosition(b, 3)

	if result.BestMove.To != SquareIndex(5, 6) {
		t.Errorf("Mate-in-1: expected move to f7, got %s", result.BestMove)
	} else {
		t.Logf("Mate-in-1: found %s ✓ (score: %s)", result.BestMove, FormatScore(result.Score))
	}

	if !isMateScore(result.Score) {
		t.Errorf("Expected mate score, got %d", result.Score)
	}
}

func TestSearchFindsObviousCapture(t *testing.T) {
	b, err := BoardFromFEN("rnb1kbnr/pppppppp/8/4q3/3P4/8/PPP1PPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	result := SearchPosition(b, 4)

	expectedFrom := SquareIndex(3, 3)
	expectedTo := SquareIndex(4, 4)

	if result.BestMove.From == expectedFrom && result.BestMove.To == expectedTo {
		t.Logf("Obvious capture: found %s ✓ (score: %s)", result.BestMove, FormatScore(result.Score))
	} else {
		if result.Score < 800 {
			t.Errorf("Expected large positive score from queen capture opportunity, got %d (move: %s)",
				result.Score, result.BestMove)
		} else {
			t.Logf("Found alternative winning move %s (score: %s) ✓", result.BestMove, FormatScore(result.Score))
		}
	}
}

func TestSearchDoesNotBlunder(t *testing.T) {
	b := NewBoard()
	result := SearchPosition(b, 4)

	if Abs(result.Score) > 100 {
		t.Errorf("Starting position search score = %d, expected roughly 0", result.Score)
	} else {
		t.Logf("Starting position: %s, score %s ✓", result.BestMove, FormatScore(result.Score))
	}
}

func TestSearchBackRankMate(t *testing.T) {
	b, err := BoardFromFEN("6k1/5ppp/8/8/8/8/8/R3K3 w Q - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	result := SearchPosition(b, 6)

	expectedTo := SquareIndex(0, 7)

	if result.BestMove.To == expectedTo {
		t.Logf("Back rank mate: found %s ✓ (score: %s)", result.BestMove, FormatScore(result.Score))
	} else {
		t.Logf("Back rank mate: engine chose %s (score: %s) — may have found alternate winning line",
			result.BestMove, FormatScore(result.Score))
		if !isMateScore(result.Score) {
			t.Errorf("Expected mate score for back rank position")
		}
	}
}
