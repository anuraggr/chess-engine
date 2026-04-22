package main

import "math/bits"

// eval fucntions doogle for doogling
const (
	UseMaterial   = true // basic piece counting
	UsePST        = true // piece-square tables
	UseMobility   = true // pseudo-legal move count bonus
	UsePawnStruct = true // doubled or isolated pawn penalties
	UseKingSafety = true // pawn shield bonus around king
)

// material value (in centipawns)
var PieceValue = [7]int{
	0,   // NoPieceType
	100, // Pawn
	320, // Knight
	330, // Bishop
	500, // Rook
	900, // Queen
	0,   // King (basically infinite)
}

// square piece table helps figuare out best square for a piece.
// (eg: queen wants to avoid corners)
var pstPawn = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, -20, -20, 10, 10, 5,
	5, -5, -10, 0, 0, -10, -5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, 5, 10, 25, 25, 10, 5, 5,
	10, 10, 20, 30, 30, 20, 10, 10,
	50, 50, 50, 50, 50, 50, 50, 50,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var pstKnight = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var pstBishop = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var pstRook = [64]int{
	0, 0, 0, 5, 5, 0, 0, 0,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	5, 10, 10, 10, 10, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var pstQueen = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var pstKingMiddle = [64]int{
	20, 30, 10, 0, 0, 10, 30, 20,
	20, 20, 0, 0, 0, 0, 20, 20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
}

var pstKingEnd = [64]int{
	-50, -30, -30, -30, -30, -30, -30, -50,
	-30, -30, 0, 0, 0, 0, -30, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -20, -10, 0, 0, -10, -20, -30,
	-50, -40, -30, -20, -20, -30, -40, -50,
}

// lookup by piece type idx.
var pstTable = [7]*[64]int{
	nil,        // NoPieceType
	&pstPawn,   // Pawn
	&pstKnight, // Knight
	&pstBishop, // Bishop
	&pstRook,   // Rook
	&pstQueen,  // Queen
	nil,        // King is handled sepretly for mid and endgame
}

var fileMask [8]uint64

func init() {
	for f := 0; f < 8; f++ {
		for r := 0; r < 8; r++ {
			fileMask[f] |= 1 << SquareIndex(f, r)
		}
	}
}

// positive -> white advantage and vice versa
func Evaluate(b *Board) int {
	score := 0

	if UseMaterial {
		score += evalMaterial(b)
	}
	if UsePST {
		score += evalPST(b)
	}
	if UseMobility {
		score += evalMobility(b)
	}
	if UsePawnStruct {
		score += evalPawnStructure(b)
	}
	if UseKingSafety {
		score += evalKingSafety(b)
	}

	return score
}

// evalMaterial counts material on both sides
func evalMaterial(b *Board) int {
	score := 0
	for pt := Pawn; pt <= Queen; pt++ {
		white := bits.OnesCount64(b.Pieces[pt] & b.Colors[White])
		black := bits.OnesCount64(b.Pieces[pt] & b.Colors[Black])
		score += (white - black) * PieceValue[pt]
	}
	return score
}

// evalPST to add piece square points.
func evalPST(b *Board) int {
	score := 0

	// we need to find if we are in endgame for kings pst.
	// endgame = nonking/nonpawn material <= 1300 (around a queen value)).
	isEndgame := b.TotalMaterial <= 1300

	for pt := Pawn; pt <= Queen; pt++ {
		tbl := pstTable[pt]
		if tbl == nil {
			continue
		}

		whites := b.Pieces[pt] & b.Colors[White]
		for whites != 0 {
			sq := PopBit(&whites)
			score += tbl[sq]
		}

		blacks := b.Pieces[pt] & b.Colors[Black]
		for blacks != 0 {
			sq := PopBit(&blacks)
			score -= tbl[sq^56] // mirror vertically for black
		}
	}

	// king PST
	wKing := b.Pieces[King] & b.Colors[White]
	if wKing != 0 {
		sq := bits.TrailingZeros64(wKing)
		if isEndgame {
			score += pstKingEnd[sq]
		} else {
			score += pstKingMiddle[sq]
		}
	}
	bKing := b.Pieces[King] & b.Colors[Black]
	if bKing != 0 {
		sq := bits.TrailingZeros64(bKing)
		if isEndgame {
			score -= pstKingEnd[sq^56]
		} else {
			score -= pstKingMiddle[sq^56]
		}
	}

	return score
}

// evalmobility gives a small bonus per attacked square.
func evalMobility(b *Board) int {
	const mobilityWeight = 2 // centipawns per attacked square

	occupied := b.Colors[White] | b.Colors[Black]

	whiteMob := countMobility(b, White, occupied)
	blackMob := countMobility(b, Black, occupied)

	return (whiteMob - blackMob) * mobilityWeight
}

// countmobility counts the number of squares attacked by a side using
func countMobility(b *Board, c Color, occupied uint64) int {
	var attacks uint64
	friendly := b.Colors[c]

	// knights
	knights := b.Pieces[Knight] & friendly
	for knights != 0 {
		sq := PopBit(&knights)
		attacks |= KnightAttacks[sq]
	}

	// bishops
	bishops := b.Pieces[Bishop] & friendly
	for bishops != 0 {
		sq := PopBit(&bishops)
		attacks |= GetBishopAttacks(sq, occupied)
	}

	// rooks
	rooks := b.Pieces[Rook] & friendly
	for rooks != 0 {
		sq := PopBit(&rooks)
		attacks |= GetRookAttacks(sq, occupied)
	}

	// queens
	queens := b.Pieces[Queen] & friendly
	for queens != 0 {
		sq := PopBit(&queens)
		attacks |= GetQueenAttacks(sq, occupied)
	}

	// minus squares occupied by friendly pieces
	attacks &^= friendly

	return bits.OnesCount64(attacks)
}

// evalPawnStructure is panelty for doubled and isolated pawns
// TODO: bonus for strong pawn structure
func evalPawnStructure(b *Board) int {
	const (
		doubledPenalty  = -10
		isolatedPenalty = -15
	)

	score := 0

	for _, c := range []Color{White, Black} {
		pawns := b.Pieces[Pawn] & b.Colors[c]
		sign := 1
		if c == Black {
			sign = -1
		}

		for f := 0; f < 8; f++ {
			pawnsOnFile := bits.OnesCount64(pawns & fileMask[f])

			// doubled pawns
			if pawnsOnFile > 1 {
				score += sign * doubledPenalty * (pawnsOnFile - 1)
			}

			// isolated pawns
			if pawnsOnFile > 0 {
				hasNeighbor := false
				if f > 0 && (pawns&fileMask[f-1]) != 0 {
					hasNeighbor = true
				}
				if f < 7 && (pawns&fileMask[f+1]) != 0 {
					hasNeighbor = true
				}
				if !hasNeighbor {
					score += sign * isolatedPenalty * pawnsOnFile
				}
			}
		}
	}

	return score
}

// evalKingSafety is for bonus for pawns shielding the king
func evalKingSafety(b *Board) int {
	const shieldBonus = 10 // per shielding pawn

	score := 0

	for _, c := range []Color{White, Black} {
		kingBB := b.Pieces[King] & b.Colors[c]
		if kingBB == 0 {
			continue
		}
		kingSq := bits.TrailingZeros64(kingBB)
		kFile := FileOf(kingSq)
		kRank := RankOf(kingSq)

		sign := 1
		pawnDir := 1
		if c == Black {
			sign = -1
			pawnDir = -1
		}

		friendlyPawns := b.Pieces[Pawn] & b.Colors[c]
		shieldCount := 0

		// check 1-2 ranks ahead on king file and adjacent files
		for df := -1; df <= 1; df++ {
			f := kFile + df
			if f < 0 || f > 7 {
				continue
			}
			for step := 1; step <= 2; step++ {
				r := kRank + pawnDir*step
				if r < 0 || r > 7 {
					continue
				}
				if HasBit(friendlyPawns, SquareIndex(f, r)) {
					shieldCount++
				}
			}
		}

		score += sign * shieldBonus * shieldCount
	}

	return score
}
