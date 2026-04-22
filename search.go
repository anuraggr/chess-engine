package main

import (
	"fmt"
	"math/bits"
	"sync/atomic"
	"time"
)

const (
	MaxPly    = 100
	MateScore = 100000
	Infinity  = MateScore + 1
)

// searchInfo carries configuration and stop signalling for a search.
type SearchInfo struct {
	Nodes    int64         // node counter
	StopFlag int32         // set to 1 to abort
	MaxDepth int           // depth limit (0 = no limit)
	MoveTime time.Duration // time limit per move (0 = no limit)
	Start    time.Time     // search started time

	PVTable  [MaxPly][MaxPly]Move
	PVLength [MaxPly]int
}

func (si *SearchInfo) Stopped() bool {
	return atomic.LoadInt32(&si.StopFlag) != 0
}

func (si *SearchInfo) Stop() {
	atomic.StoreInt32(&si.StopFlag, 1)
}

func (si *SearchInfo) IncNodes() {
	atomic.AddInt64(&si.Nodes, 1)
}

func (si *SearchInfo) CheckTimeLimit() {
	if si.MoveTime > 0 && time.Since(si.Start) >= si.MoveTime {
		si.Stop()
	}
}

// move ordering to score moves for better pruning
// MVV LVA
var mvvLVA [7][7]int

func init() {
	victimScore := [7]int{0, 10, 20, 30, 40, 50, 60}
	attackerScore := [7]int{0, 6, 5, 4, 3, 2, 1}

	for victim := NoPieceType; victim <= King; victim++ {
		for attacker := NoPieceType; attacker <= King; attacker++ {
			mvvLVA[victim][attacker] = victimScore[victim]*10 + attackerScore[attacker]
		}
	}
}

func scoreMove(b *Board, m Move, ttMove Move) int {
	if m == ttMove {
		return 1000000
	}
	score := 0
	if m.Flag == FlagPromotion {
		score += 800 + PieceValue[m.Promotion]
	}
	capturedPiece := b.PieceAt(m.To)
	if capturedPiece != NoPiece {
		movingPiece := b.PieceAt(m.From)
		score += mvvLVA[capturedPiece.Type()][movingPiece.Type()] * 10
	}
	if m.Flag == FlagEnPassant {
		score += mvvLVA[Pawn][Pawn] * 10
	}
	return score
}

func sortMoves(b *Board, moves []Move, ttMove Move) {
	n := len(moves)
	scores := make([]int, n)
	for i, m := range moves {
		scores[i] = scoreMove(b, m, ttMove)
	}
	for i := 1; i < n; i++ {
		key := scores[i]
		keyMove := moves[i]
		j := i - 1
		for j >= 0 && scores[j] < key {
			scores[j+1] = scores[j]
			moves[j+1] = moves[j]
			j--
		}
		scores[j+1] = key
		moves[j+1] = keyMove
	}
}

// Quiescence search - continue search if pos unstable
func quiescence(b *Board, alpha, beta int, si *SearchInfo) int {
	si.IncNodes()

	if si.Stopped() {
		return 0
	}

	standPat := Evaluate(b)
	if b.Turn == Black {
		standPat = -standPat
	}

	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	moves := GenerateLegalMoves(b)
	sortMoves(b, moves, Move{})

	for _, m := range moves {
		isCapture := b.PieceAt(m.To) != NoPiece || m.Flag == FlagEnPassant
		isPromo := m.Flag == FlagPromotion
		if !isCapture && !isPromo {
			continue
		}

		nb := MakeMove(b, m)
		score := -quiescence(nb, -beta, -alpha, si)

		if si.Stopped() {
			return 0
		}

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}

// negamax with alpha-beta pruning
func negamax(b *Board, depth, alpha, beta, ply int, si *SearchInfo) int {
	if ply < MaxPly {
		si.PVLength[ply] = ply
	}
	si.IncNodes()

	// we check time limit every 2048 nodes
	if atomic.LoadInt64(&si.Nodes)&2047 == 0 {
		si.CheckTimeLimit()
	}

	if si.Stopped() {
		return 0
	}

	if b.HalfMoveClock >= 100 || isInsufficientMaterial(b) || isRepetition(b) {
		return 0
	}

	var ttMove Move
	if entry, found := TT.Probe(b.Hash); found {
		ttMove = entry.BestMove.Unpack()
		if int(entry.Depth) >= depth {
			score := int(entry.Score)
			if entry.Flag == FlagExact {
				return score
			}
			if entry.Flag == FlagAlpha && score <= alpha {
				return alpha
			}
			if entry.Flag == FlagBeta && score >= beta {
				return beta
			}
		}
	}

	const R = 2

	if depth >= 3 && !IsInCheck(b, b.Turn) {
		nb := MakeNullMove(b)
		nullScore := -negamax(nb, depth-1-R, -beta, -beta+1, ply+1, si)

		if si.Stopped() {
			return 0
		}

		if nullScore >= beta {
			return beta
		}
	}

	legalMoves := GenerateLegalMoves(b)

	if len(legalMoves) == 0 {
		if IsInCheck(b, b.Turn) {
			return -(MateScore - depth)
		}
		return 0
	}

	if depth <= 0 {
		return quiescence(b, alpha, beta, si)
	}

	sortMoves(b, legalMoves, ttMove)

	bestScore := -Infinity
	var bestMove Move

	originalAlpha := alpha

	for _, m := range legalMoves {
		nb := MakeMove(b, m)
		score := -negamax(nb, depth-1, -beta, -alpha, ply+1, si)

		if si.Stopped() {
			return 0
		}

		if score > bestScore {
			bestScore = score
			bestMove = m
		}
		if score > alpha {
			alpha = score

			if ply < MaxPly {
				si.PVTable[ply][ply] = m
				si.PVLength[ply] = ply + 1

				if ply+1 < MaxPly {
					length := si.PVLength[ply+1]
					if length > ply+1 {
						copy(si.PVTable[ply][ply+1:length], si.PVTable[ply+1][ply+1:length])
					}
					si.PVLength[ply] = length
				}
			}
		}
		if alpha >= beta {
			break
		}
	}

	if !si.Stopped() {
		flag := FlagAlpha // assume we failed low
		if bestScore >= beta {
			flag = FlagBeta // beta cutoff (lower bound)
		} else if bestScore > originalAlpha {
			flag = FlagExact // we improved alpha without hitting beta
		}

		TT.Store(b.Hash, uint8(depth), int16(bestScore), flag, bestMove)
	}

	return bestScore
}

// SearchPosition with iterative deepening
type SearchResult struct {
	BestMove Move
	Score    int
	Depth    int
}

// defaults (no stop, depth limit only).
func SearchPosition(b *Board, maxDepth int) SearchResult {
	si := &SearchInfo{MaxDepth: maxDepth, Start: time.Now()}
	return SearchPositionWithInfo(b, si)
}

// runs an interruptible iterative deepening search.
func SearchPositionWithInfo(b *Board, si *SearchInfo) SearchResult {
	var best SearchResult

	maxD := si.MaxDepth
	if maxD <= 0 {
		maxD = 100 // infinite
	}

	for d := 1; d <= maxD; d++ {
		if si.Stopped() {
			break
		}

		alpha := -Infinity
		beta := Infinity

		legalMoves := GenerateLegalMoves(b)
		sortMoves(b, legalMoves, best.BestMove)

		// move ordering
		// if best.Depth > 0 && best.BestMove.From != best.BestMove.To {
		// 	for i, m := range legalMoves {
		// 		if m == best.BestMove {
		// 			legalMoves[0], legalMoves[i] = legalMoves[i], legalMoves[0]
		// 			break
		// 		}
		// 	}
		// }
		//I dont think I need all thsi

		si.PVLength[0] = 0 // Initialize at start of depth

		bestScore := -Infinity
		var bestMove Move

		for _, m := range legalMoves {
			nb := MakeMove(b, m)
			score := -negamax(nb, d-1, -beta, -alpha, 1, si)

			if si.Stopped() {
				break
			}

			if score > bestScore {
				bestScore = score
				bestMove = m
			}
			if score > alpha {
				alpha = score

				si.PVTable[0][0] = m
				si.PVLength[0] = 1

				length := si.PVLength[1]
				if length > 1 {
					copy(si.PVTable[0][1:length], si.PVTable[1][1:length])
				}
				si.PVLength[0] = length
			}
		}

		// we only update best if we completed this depth without being stopped
		if !si.Stopped() {
			best = SearchResult{BestMove: bestMove, Score: bestScore, Depth: d}

			//store root evaluation in TT to make it seen by ExtractPV
			TT.Store(b.Hash, uint8(d), int16(bestScore), FlagExact, bestMove)

			// prints UCI info line
			elapsed := time.Since(si.Start)
			ms := elapsed.Milliseconds()
			if ms == 0 {
				ms = 1
			}
			nps := atomic.LoadInt64(&si.Nodes) * 1000 / ms

			scoreStr := fmt.Sprintf("cp %d", best.Score)
			if best.Score > MateScore-500 {
				plies := MateScore - best.Score
				scoreStr = fmt.Sprintf("mate %d", (plies+1)/2)
			} else if best.Score < -MateScore+500 {
				plies := MateScore + best.Score
				scoreStr = fmt.Sprintf("mate -%d", (plies+1)/2)
			}

			pvString := ""
			for i := 0; i < si.PVLength[0]; i++ {
				if i > 0 {
					pvString += " "
				}
				pvString += si.PVTable[0][i].String()
			}
			if si.PVLength[0] == 0 {
				pvString = best.BestMove.String()
			}

			fmt.Printf("info depth %d score %s nodes %d nps %d time %d pv %s\n",
				d, scoreStr, atomic.LoadInt64(&si.Nodes), nps, elapsed.Milliseconds(), pvString)

			if isMateScore(bestScore) {
				break
			}
		}
	}

	return best
}

// helpers
func isMateScore(score int) bool {
	return score > MateScore-500 || score < -MateScore+500
}

func FormatScore(score int) string {
	if score > MateScore-500 {
		plies := MateScore - score
		moves := (plies + 1) / 2
		return fmt.Sprintf("mate in %d", moves)
	}
	if score < -MateScore+500 {
		plies := MateScore + score
		moves := (plies + 1) / 2
		return fmt.Sprintf("mated in %d", moves)
	}
	return fmt.Sprintf("%.2f", float64(score)/100.0)
}

func CountMaterial(b *Board, c Color) int {
	mat := 0
	for pt := Knight; pt <= Queen; pt++ {
		mat += bits.OnesCount64(b.Pieces[pt]&b.Colors[c]) * PieceValue[pt]
	}
	return mat
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
