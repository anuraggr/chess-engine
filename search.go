package main

import (
	"fmt"
	"math/bits"
	"sync/atomic"
	"time"
)

const (
	MaxPly    = 100
	MateScore = 30000
	Infinity  = 32000
)

// searchInfo carries configuration and stop signalling for a search.
type SearchInfo struct {
	Nodes    int64         // node counter
	StopFlag int32         // set to 1 to abort
	MaxDepth int           // depth limit (0 = no limit)
	MoveTime time.Duration // time limit per move (0 = no limit)
	Start    time.Time     // search started time

	MoveLists [MaxPly]MoveList
	PVTable   [MaxPly][MaxPly]Move
	PVLength  [MaxPly]int
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

func sortMoves(b *Board, ml *MoveList, ttMove Move) {
	n := ml.Count
	for i := 0; i < n; i++ {
		ml.Scores[i] = scoreMove(b, ml.Moves[i], ttMove)
	}
	for i := 1; i < n; i++ {
		keyScore := ml.Scores[i]
		keyMove := ml.Moves[i]
		j := i - 1
		for j >= 0 && ml.Scores[j] < keyScore {
			ml.Scores[j+1] = ml.Scores[j]
			ml.Moves[j+1] = ml.Moves[j]
			j--
		}
		ml.Scores[j+1] = keyScore
		ml.Moves[j+1] = keyMove
	}
}

// Quiescence search - continue search if pos unstable
func quiescence(b *Board, alpha, beta, ply int, si *SearchInfo) int {
	if ply >= MaxPly {
		return Evaluate(b)
	}
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

	ml := &si.MoveLists[ply]
	GeneratePseudoLegalMoves(b, ml)
	sortMoves(b, ml, Move{})

	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		isCapture := b.PieceAt(m.To) != NoPiece || m.Flag == FlagEnPassant
		isPromo := m.Flag == FlagPromotion
		if !isCapture && !isPromo {
			continue
		}

		info := MakeMove(b, m)
		if IsInCheck(b, b.Turn.Other()) {
			UnmakeMove(b, m, info)
			continue
		}

		score := -quiescence(b, -beta, -alpha, ply+1, si)
		UnmakeMove(b, m, info)

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

	inCheck := IsInCheck(b, b.Turn)

	if depth >= 3 && !inCheck {
		info := MakeNullMove(b)
		nullScore := -negamax(b, depth-1-R, -beta, -beta+1, ply+1, si)
		UnmakeNullMove(b, info)

		if si.Stopped() {
			return 0
		}

		if nullScore >= beta {
			return beta
		}
	}

	ml := &si.MoveLists[ply]
	GeneratePseudoLegalMoves(b, ml)

	if depth <= 0 {
		return quiescence(b, alpha, beta, ply, si)
	}

	sortMoves(b, ml, ttMove)

	bestScore := -Infinity
	var bestMove Move

	originalAlpha := alpha
	movesEvaluated := 0 // move counter for LMR
	legalMovesCount := 0

	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		isCapture := b.PieceAt(m.To) != NoPiece || m.Flag == FlagEnPassant
		isPromo := m.Flag == FlagPromotion

		info := MakeMove(b, m)
		if IsInCheck(b, b.Turn.Other()) {
			UnmakeMove(b, m, info)
			continue
		}
		legalMovesCount++

		movesEvaluated++
		var score int
		needsFullSearch := true

		inCheckPost := IsInCheck(b, b.Turn)
		if depth >= 3 && movesEvaluated > 4 && !isCapture && !isPromo && !inCheck && !inCheckPost {

			reduction := 1
			if depth > 4 && movesEvaluated > 10 {
				reduction = 2
			}
			score = -negamax(b, depth-1-reduction, -alpha-1, -alpha, ply+1, si)
			if score > alpha {
				needsFullSearch = true
			} else {
				needsFullSearch = false
			}
		}

		if needsFullSearch {
			score = -negamax(b, depth-1, -beta, -alpha, ply+1, si)
		}

		UnmakeMove(b, m, info)

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

	if legalMovesCount == 0 {
		if inCheck {
			return -(MateScore - depth)
		}
		return 0
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

func SearchPositionWithInfo(b *Board, si *SearchInfo) SearchResult {
	if bookMove, found := GetBookMove(b); found {
		fmt.Printf("info depth 1 score cp 0 time 0 pv %s\n", bookMove.String())
		return SearchResult{BestMove: bookMove, Score: 0, Depth: 1}
	}

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

		ml := &si.MoveLists[0]
		GeneratePseudoLegalMoves(b, ml)
		sortMoves(b, ml, best.BestMove)

		// move ordering
		// if best.Depth > 0 && best.BestMove.From != best.BestMove.To {
		// 	for i := 0; i < ml.Count; i++ {
		// 		m := ml.Moves[i]
		// 		if m == best.BestMove {
		// 			ml.Moves[0], ml.Moves[i] = ml.Moves[i], ml.Moves[0]
		// 			break
		// 		}
		// 	}
		// }
		//I dont think I need all thsi

		si.PVLength[0] = 0 // Initialize at start of depth

		bestScore := -Infinity
		var bestMove Move

		for i := 0; i < ml.Count; i++ {
			m := ml.Moves[i]
			info := MakeMove(b, m)
			if IsInCheck(b, b.Turn.Other()) {
				UnmakeMove(b, m, info)
				continue
			}

			score := -negamax(b, d-1, -beta, -alpha, 1, si)
			UnmakeMove(b, m, info)

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
