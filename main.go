package main

import (
	"fmt"
	"os"
	"time"
)

func DebugBlunder() {
	b, _ := BoardFromFEN("6k1/5rp1/1P6/3Q4/5P2/6K1/p5PP/r7 b - - 0 51")

	moves := GenerateLegalMoves(b)

	d := 15

	fmt.Printf("--- DEPTH %d MOVE EVALUATIONS --- \n", d)
	for _, m := range moves {
		info := MakeMove(b, m)

		si := &SearchInfo{MaxDepth: 7, Start: time.Now()}
		score := -negamax(b, d, -MateScore, MateScore, 1, si)

		pvString := ""
		for i := 1; i < si.PVLength[1]; i++ {
			pvString += si.PVTable[1][i].String() + " "
		}

		fmt.Printf("Move: %s | Score: %d\n", m.String(), score)
		fmt.Printf("True PV: %s\n", pvString)
		UnmakeMove(b, m, info)
	}
	fmt.Println("END")
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--self-play" {
		selfPlay()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--debug" {
		DebugBlunder()
	}

	err := LoadBook("/home/anuragrai/Documents/chess-engine/Titans.bin")
	if err != nil {
		fmt.Println("info string Warning: Could not load opening book")
	} else {
		fmt.Println("info string Opening book loaded successfully")
	}

	UCI()
}

func selfPlay() {
	board := NewBoard()
	moveCount := 0
	searchDepth := 5

	fmt.Println(board)
	fmt.Printf("Search depth: %d\n\n", searchDepth)

	for {
		outcome, method := GetOutcome(board)
		if outcome != NoOutcome {
			fmt.Println(board)
			fmt.Printf("Game completed. %s by %s.\n", outcome, method)
			fmt.Printf("Total moves: %d\n", moveCount)
			break
		}

		result := SearchPosition(board, searchDepth)
		fmt.Printf("%d. %s  %s  (depth %d, eval %s)\n",
			board.FullMoveNumber,
			board.Turn,
			result.BestMove,
			result.Depth,
			FormatScore(result.Score),
		)

		MakeMove(board, result.BestMove)
		moveCount++
	}
}
