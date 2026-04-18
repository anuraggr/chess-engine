package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--self-play" {
		selfPlay()
		return
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

		board = MakeMove(board, result.BestMove)
		moveCount++
	}
}
