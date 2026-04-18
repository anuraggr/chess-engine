package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	EngineName   = "ChessEngine"
	EngineAuthor = "Anurag"
)

// the UCI protocol loop reading from stdin
func UCI() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	board := NewBoard()
	var currentSearch *SearchInfo
	var searchWg sync.WaitGroup

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		tokens := strings.Fields(line)
		cmd := tokens[0]

		switch cmd {
		case "uci":
			fmt.Printf("id name %s\n", EngineName)
			fmt.Printf("id author %s\n", EngineAuthor)
			// I don't know why but leetcode-bot seems to require these options
			fmt.Println("option name Move Overhead type spin default 500 min 0 max 10000")
			fmt.Println("option name Threads type spin default 1 min 1 max 128")
			fmt.Println("option name Hash type spin default 16 min 1 max 1024")
			fmt.Println("option name SyzygyPath type string default <empty>")
			fmt.Println("option name UCI_ShowWDL type check default false")

			fmt.Println("uciok")

		case "isready":
			fmt.Println("readyok")

		// case "setoption":
		// 	// Safely ignore option configuration
		// 	continue

		case "ucinewgame":
			board = NewBoard()

		case "position":
			board = parsePosition(tokens[1:])

		case "go":
			if currentSearch != nil {
				currentSearch.Stop()
			}
			searchWg.Wait() // wait for previous search goroutine to finish
			currentSearch = &SearchInfo{Start: time.Now()}
			parseGo(tokens[1:], currentSearch, board, &searchWg)

		case "stop":
			if currentSearch != nil {
				currentSearch.Stop()
			}

		case "quit":
			if currentSearch != nil {
				currentSearch.Stop()
			}
			searchWg.Wait()
			return

		// debug helpers
		case "d":
			fmt.Println(board)
			fmt.Println("FEN:", board.FEN())

		case "eval":
			score := Evaluate(board)
			fmt.Printf("Eval: %d cp\n", score)
		}
	}

	// stdin closed (EOF?) -> wait for any running search before quiting
	if currentSearch != nil {
		currentSearch.Stop()
	}
	searchWg.Wait()
}

// position startpos [moves e2e4 e7e5 ...]
// position fen <fen> [moves e2e4 ...]
func parsePosition(tokens []string) *Board {
	if len(tokens) == 0 {
		//we probably would never get a empty tokens array until gui is broken
		//so this is kinda redundant but still a guardrail
		return NewBoard()
	}

	var board *Board
	movesIdx := -1

	if tokens[0] == "startpos" {
		board = NewBoard()
		// look for "moves" keyword. in theory, if it exists, it should
		// always we the one after "startpos" so we can use if-stmt
		// but this is more defensive
		for i, t := range tokens {
			if t == "moves" {
				movesIdx = i + 1
				break
			}
		}
	} else if tokens[0] == "fen" {
		// Collect FEN fields (up to 6 fields, stopping at "moves")
		fenParts := []string{}
		i := 1
		for i < len(tokens) && tokens[i] != "moves" {
			fenParts = append(fenParts, tokens[i])
			i++
		}
		fen := strings.Join(fenParts, " ")
		var err error
		board, err = BoardFromFEN(fen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "info string invalid FEN: %v\n", err)
			return NewBoard()
		}
		if i < len(tokens) && tokens[i] == "moves" {
			movesIdx = i + 1
		}
	} else {
		return NewBoard()
	}

	// apply moves
	if movesIdx >= 0 {
		for _, moveStr := range tokens[movesIdx:] {
			move, err := ParseUCIMove(board, moveStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "info string invalid move %s: %v\n", moveStr, err)
				break
			}
			board = MakeMove(board, move)
		}
	}

	return board
}

// go depth 6 (the uci might provide depth depending on mode)
// go movetime 5000
// go wtime 300000 btime 300000 [winc 0 binc 0]
// go infinite
func parseGo(tokens []string, si *SearchInfo, board *Board, wg *sync.WaitGroup) {
	depth := 0
	moveTime := time.Duration(0)
	wtime, btime := time.Duration(0), time.Duration(0)
	winc, binc := time.Duration(0), time.Duration(0)
	infinite := false
	movestogo := 30 // default estimate

	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "depth":
			if i+1 < len(tokens) {
				depth, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "movetime":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				moveTime = time.Duration(ms) * time.Millisecond
				i++
			}
		case "wtime":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				wtime = time.Duration(ms) * time.Millisecond
				i++
			}
		case "btime":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				btime = time.Duration(ms) * time.Millisecond
				i++
			}
		case "winc":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				winc = time.Duration(ms) * time.Millisecond
				i++
			}
		case "binc":
			if i+1 < len(tokens) {
				ms, _ := strconv.Atoi(tokens[i+1])
				binc = time.Duration(ms) * time.Millisecond
				i++
			}
		case "movestogo":
			if i+1 < len(tokens) {
				movestogo, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "infinite":
			infinite = true
		}
	}

	if depth > 0 {
		si.MaxDepth = depth
	} else if moveTime > 0 {
		si.MoveTime = moveTime
		si.MaxDepth = 100 // effectively unlimited
	} else if wtime > 0 || btime > 0 {
		// allocate a fraction of remaining time
		var remaining, inc time.Duration
		if board.Turn == White {
			remaining = wtime
			inc = winc
		} else {
			remaining = btime
			inc = binc
		}

		// Use 1/movestogo of remaining time + most of increment
		allocated := remaining/time.Duration(movestogo) + inc*8/10

		// safety: never use more than 50% of remaining time
		maxTime := remaining / 2
		if allocated > maxTime {
			allocated = maxTime
		}
		// floor at 100ms
		if allocated < 100*time.Millisecond {
			allocated = 100 * time.Millisecond
		}

		si.MoveTime = allocated
		si.MaxDepth = 100
	} else if infinite {
		si.MaxDepth = 100
	} else {
		// Default: depth 6
		si.MaxDepth = 6
	}

	// run search in a goroutine so we can receive "stop" while searching
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := SearchPositionWithInfo(board, si)
		fmt.Printf("bestmove %s\n", result.BestMove)
	}()
}

// parseUCIMove — parse "e2e4", "e7e8q" etc. into a Move
func ParseUCIMove(b *Board, s string) (Move, error) {
	if len(s) < 4 || len(s) > 5 {
		return Move{}, fmt.Errorf("invalid move string: %s", s)
	}

	from, err := SquareFromAlgebraic(s[0:2])
	if err != nil {
		return Move{}, err
	}
	to, err := SquareFromAlgebraic(s[2:4])
	if err != nil {
		return Move{}, err
	}

	var promo PieceType
	if len(s) == 5 {
		switch s[4] {
		case 'q':
			promo = Queen
		case 'r':
			promo = Rook
		case 'b':
			promo = Bishop
		case 'n':
			promo = Knight
		default:
			return Move{}, fmt.Errorf("invalid promotion: %c", s[4])
		}
	}

	// natch against legal moves to get the correct flags
	legalMoves := GenerateLegalMoves(b)
	for _, m := range legalMoves {
		if m.From == from && m.To == to {
			if promo != NoPieceType {
				if m.Promotion == promo {
					return m, nil
				}
			} else if m.Flag != FlagPromotion {
				return m, nil
			}
		}
	}

	return Move{}, fmt.Errorf("illegal move: %s", s)
}
