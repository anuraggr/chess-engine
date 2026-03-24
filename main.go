package main

import (
	"fmt"
	"math/rand"
)

func main() {
	board := NewBoard()
	moveCount := 0

	for {
		outcome, method := GetOutcome(board)
		if outcome != NoOutcome {
			fmt.Println(board)
			fmt.Printf("Game completed. %s by %s.\n", outcome, method)
			fmt.Printf("Total moves: %d\n", moveCount)
			break
		}

		moves := GenerateLegalMoves(board)
		move := moves[rand.Intn(len(moves))]
		board = MakeMove(board, move)
		moveCount++
	}
}
