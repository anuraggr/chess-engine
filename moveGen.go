package main

import "math/bits"

// Move flags
type MoveFlag uint8

const (
	FlagNone        MoveFlag = iota
	FlagDoublePush           // pawn double push
	FlagEnPassant            // en passant capture
	FlagCastleKing           // kingside castle
	FlagCastleQueen          // queenside castle
	FlagPromotion            // promotion (check Promotion field)
)

// Move
type Move struct {
	From      int
	To        int
	Promotion PieceType // NoPieceType if no promotion
	Flag      MoveFlag
}

func (m Move) String() string {
	s := SquareToAlgebraic(m.From) + SquareToAlgebraic(m.To)
	if m.Promotion != NoPieceType {
		switch m.Promotion {
		case Knight:
			s += "n"
		case Bishop:
			s += "b"
		case Rook:
			s += "r"
		case Queen:
			s += "q"
		}
	}
	return s
}

// Precalculated Attack Tables
var KnightAttacks [64]uint64
var KingAttacks [64]uint64
var PawnAttacks [2][64]uint64 // [color][sq] attacks for pawn on sq

// Rays[sq][dir]
var Rays [64][8]uint64

const (
	//Directions are used for precalculating bishop, rook, queen moves
	DirN = iota
	DirS
	DirE
	DirW
	DirNE
	DirNW
	DirSE
	DirSW
)

func init() {
	for sq := 0; sq < 64; sq++ {
		file := FileOf(sq)
		rank := RankOf(sq)

		//Knight attack sq precalculated from every sq on board
		//KnightAttack[27] (knight on 27th square) gives a3,a5, and so on.
		for _, off := range [8][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}} {
			nf, nr := file+off[0], rank+off[1]
			if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
				SetBit(&KnightAttacks[sq], SquareIndex(nf, nr))
			}
		}

		//Similarly with the king
		for _, off := range [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}} {
			nf, nr := file+off[0], rank+off[1]
			if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
				SetBit(&KingAttacks[sq], SquareIndex(nf, nr))
			}
		}

		//Pawns are little different. White and black pawns move in different directions
		if rank < 7 && file > 0 {
			SetBit(&PawnAttacks[White][sq], SquareIndex(file-1, rank+1))
		}
		if rank < 7 && file < 7 {
			SetBit(&PawnAttacks[White][sq], SquareIndex(file+1, rank+1))
		}
		if rank > 0 && file > 0 {
			SetBit(&PawnAttacks[Black][sq], SquareIndex(file-1, rank-1))
		}
		if rank > 0 && file < 7 {
			SetBit(&PawnAttacks[Black][sq], SquareIndex(file+1, rank-1))
		}

		//Rays are for Bishops, queens, rooks
		for r := rank + 1; r <= 7; r++ {
			SetBit(&Rays[sq][DirN], SquareIndex(file, r))
		}
		for r := rank - 1; r >= 0; r-- {
			SetBit(&Rays[sq][DirS], SquareIndex(file, r))
		}
		for f := file + 1; f <= 7; f++ {
			SetBit(&Rays[sq][DirE], SquareIndex(f, rank))
		}
		for f := file - 1; f >= 0; f-- {
			SetBit(&Rays[sq][DirW], SquareIndex(f, rank))
		}
		for r, f := rank+1, file+1; r <= 7 && f <= 7; r, f = r+1, f+1 {
			SetBit(&Rays[sq][DirNE], SquareIndex(f, r))
		}
		for r, f := rank+1, file-1; r <= 7 && f >= 0; r, f = r+1, f-1 {
			SetBit(&Rays[sq][DirNW], SquareIndex(f, r))
		}
		for r, f := rank-1, file+1; r >= 0 && f <= 7; r, f = r-1, f+1 {
			SetBit(&Rays[sq][DirSE], SquareIndex(f, r))
		}
		for r, f := rank-1, file-1; r >= 0 && f >= 0; r, f = r-1, f-1 {
			SetBit(&Rays[sq][DirSW], SquareIndex(f, r))
		}
	}
}

// Sliding Attack Generation
func getPositiveRayAttacks(sq int, dir int, occupied uint64) uint64 {
	//positive for where sq index increases
	//we can get squares where a piece is blocking with a AND operation with occupied bitboard
	ray := Rays[sq][dir]
	blocker := ray & occupied
	if blocker != 0 {
		//since this is positive, the closest blocker is one with lowest sq idx
		blockSq := bits.TrailingZeros64(blocker)
		//chop off anything beyond blocker with this XOR operation
		ray ^= Rays[blockSq][dir]
	}
	return ray
}

func getNegativeRayAttacks(sq int, dir int, occupied uint64) uint64 {
	//negative for where sq index decreases
	ray := Rays[sq][dir]
	blocker := ray & occupied
	if blocker != 0 {
		//since this is opposite of positive, blocker is last one from start
		blockSq := 63 - bits.LeadingZeros64(blocker)
		ray ^= Rays[blockSq][dir]
	}
	return ray
}

func GetBishopAttacks(sq int, occupied uint64) uint64 {
	return getPositiveRayAttacks(sq, DirNE, occupied) |
		getPositiveRayAttacks(sq, DirNW, occupied) |
		getNegativeRayAttacks(sq, DirSE, occupied) |
		getNegativeRayAttacks(sq, DirSW, occupied)
}

func GetRookAttacks(sq int, occupied uint64) uint64 {
	return getPositiveRayAttacks(sq, DirN, occupied) |
		getPositiveRayAttacks(sq, DirE, occupied) |
		getNegativeRayAttacks(sq, DirS, occupied) |
		getNegativeRayAttacks(sq, DirW, occupied)
}

func GetQueenAttacks(sq int, occupied uint64) uint64 {
	return GetBishopAttacks(sq, occupied) | GetRookAttacks(sq, occupied)
}

// Attack detection
func isSquareAttacked(b *Board, sq int, byColor Color) bool {
	enemyPawns := b.Colors[byColor] & b.Pieces[Pawn]
	if (PawnAttacks[byColor.Other()][sq] & enemyPawns) != 0 {
		return true
	}

	enemyKnights := b.Colors[byColor] & b.Pieces[Knight]
	if (KnightAttacks[sq] & enemyKnights) != 0 {
		return true
	}

	enemyKing := b.Colors[byColor] & b.Pieces[King]
	if (KingAttacks[sq] & enemyKing) != 0 {
		return true
	}

	occupied := b.Colors[White] | b.Colors[Black]

	enemyBishopsQueens := b.Colors[byColor] & (b.Pieces[Bishop] | b.Pieces[Queen])
	if enemyBishopsQueens != 0 {
		if (GetBishopAttacks(sq, occupied) & enemyBishopsQueens) != 0 {
			return true
		}
	}

	enemyRooksQueens := b.Colors[byColor] & (b.Pieces[Rook] | b.Pieces[Queen])
	if enemyRooksQueens != 0 {
		if (GetRookAttacks(sq, occupied) & enemyRooksQueens) != 0 {
			return true
		}
	}

	return false
}

func IsInCheck(b *Board, color Color) bool {
	kingBB := b.Colors[color] & b.Pieces[King]
	if kingBB == 0 {
		return false // Should never happen
	}
	kingSq := bits.TrailingZeros64(kingBB)
	return isSquareAttacked(b, kingSq, color.Other())
}

// Pseudo-legal move generators
// movinga piece might result in check on ur king. thats a illegal move
// we filter it out in GenerateLegalMove
// generate then filter is std pattern because its very fast and only
// very little moves are illegal
func appendPromotions(moves []Move, from, to int) []Move {
	for _, pt := range []PieceType{Queen, Rook, Bishop, Knight} {
		moves = append(moves, Move{From: from, To: to, Promotion: pt, Flag: FlagPromotion})
	}
	return moves
}

func GeneratePseudoLegalMoves(b *Board) []Move {
	moves := make([]Move, 0, 64)
	color := b.Turn
	occupied := b.Colors[White] | b.Colors[Black]
	empty := ^occupied
	enemyOrEmpty := ^b.Colors[color]

	// panws
	pawns := b.Colors[color] & b.Pieces[Pawn]
	if color == White {
		// single push
		pushed := (pawns << 8) & empty
		bb := pushed
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to - 8
			if to >= 56 {
				moves = appendPromotions(moves, from, to)
			} else {
				moves = append(moves, Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		// double push
		doublePushed := ((pushed & 0x0000000000FF0000) << 8) & empty
		bb = doublePushed
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to - 16
			moves = append(moves, Move{From: from, To: to, Flag: FlagDoublePush})
			bb &= bb - 1
		}
		// captures left
		capsL := (pawns << 7) & 0x7F7F7F7F7F7F7F7F & b.Colors[Black]
		bb = capsL
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to - 7
			if to >= 56 {
				moves = appendPromotions(moves, from, to)
			} else {
				moves = append(moves, Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		// captures right
		capsR := (pawns << 9) & 0xFEFEFEFEFEFEFEFE & b.Colors[Black]
		bb = capsR
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to - 9
			if to >= 56 {
				moves = appendPromotions(moves, from, to)
			} else {
				moves = append(moves, Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		// en passant
		if b.EnPassantSquare >= 0 {
			epSq := int(b.EnPassantSquare)
			caps := PawnAttacks[Black][epSq] & pawns
			bb = caps
			for bb != 0 {
				from := bits.TrailingZeros64(bb)
				moves = append(moves, Move{From: from, To: epSq, Flag: FlagEnPassant})
				bb &= bb - 1
			}
		}
	} else {
		//same but for black
		pushed := (pawns >> 8) & empty
		bb := pushed
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 8
			if to <= 7 {
				moves = appendPromotions(moves, from, to)
			} else {
				moves = append(moves, Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		doublePushed := ((pushed & 0x0000FF0000000000) >> 8) & empty
		bb = doublePushed
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 16
			moves = append(moves, Move{From: from, To: to, Flag: FlagDoublePush})
			bb &= bb - 1
		}
		capsR := (pawns >> 7) & 0xFEFEFEFEFEFEFEFE & b.Colors[White]
		bb = capsR
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 7
			if to <= 7 {
				moves = appendPromotions(moves, from, to)
			} else {
				moves = append(moves, Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		capsL := (pawns >> 9) & 0x7F7F7F7F7F7F7F7F & b.Colors[White]
		bb = capsL
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 9
			if to <= 7 {
				moves = appendPromotions(moves, from, to)
			} else {
				moves = append(moves, Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		if b.EnPassantSquare >= 0 {
			epSq := int(b.EnPassantSquare)
			caps := PawnAttacks[White][epSq] & pawns
			bb = caps
			for bb != 0 {
				from := bits.TrailingZeros64(bb)
				moves = append(moves, Move{From: from, To: epSq, Flag: FlagEnPassant})
				bb &= bb - 1
			}
		}
	}

	// knights
	knights := b.Colors[color] & b.Pieces[Knight]
	for knights != 0 {
		from := bits.TrailingZeros64(knights)
		attacks := KnightAttacks[from] & enemyOrEmpty
		for attacks != 0 {
			to := bits.TrailingZeros64(attacks)
			moves = append(moves, Move{From: from, To: to})
			attacks &= attacks - 1
		}
		knights &= knights - 1
	}

	// bishiops
	bishops := b.Colors[color] & b.Pieces[Bishop]
	for bishops != 0 {
		from := bits.TrailingZeros64(bishops)
		attacks := GetBishopAttacks(from, occupied) & enemyOrEmpty
		for attacks != 0 {
			to := bits.TrailingZeros64(attacks)
			moves = append(moves, Move{From: from, To: to})
			attacks &= attacks - 1
		}
		bishops &= bishops - 1
	}

	// rooks
	rooks := b.Colors[color] & b.Pieces[Rook]
	for rooks != 0 {
		from := bits.TrailingZeros64(rooks)
		attacks := GetRookAttacks(from, occupied) & enemyOrEmpty
		for attacks != 0 {
			to := bits.TrailingZeros64(attacks)
			moves = append(moves, Move{From: from, To: to})
			attacks &= attacks - 1
		}
		rooks &= rooks - 1
	}

	// queens
	queens := b.Colors[color] & b.Pieces[Queen]
	for queens != 0 {
		from := bits.TrailingZeros64(queens)
		attacks := GetQueenAttacks(from, occupied) & enemyOrEmpty
		for attacks != 0 {
			to := bits.TrailingZeros64(attacks)
			moves = append(moves, Move{From: from, To: to})
			attacks &= attacks - 1
		}
		queens &= queens - 1
	}

	// kings
	kings := b.Colors[color] & b.Pieces[King]
	if kings != 0 {
		from := bits.TrailingZeros64(kings)
		attacks := KingAttacks[from] & enemyOrEmpty
		for attacks != 0 {
			to := bits.TrailingZeros64(attacks)
			moves = append(moves, Move{From: from, To: to})
			attacks &= attacks - 1
		}
	}

	moves = generateCastlingMoves(b, moves)
	return moves
}

func generateCastlingMoves(b *Board, moves []Move) []Move {
	color := b.Turn
	enemy := color.Other()
	occupied := b.Colors[White] | b.Colors[Black]

	if color == White {
		if b.CastlingRights&CastleWhiteKing != 0 && HasBit(b.Colors[White]&b.Pieces[Rook], 7) && HasBit(b.Colors[White]&b.Pieces[King], 4) {
			if !HasBit(occupied, 5) && !HasBit(occupied, 6) {
				if !isSquareAttacked(b, 4, enemy) && !isSquareAttacked(b, 5, enemy) && !isSquareAttacked(b, 6, enemy) {
					moves = append(moves, Move{From: 4, To: 6, Flag: FlagCastleKing})
				}
			}
		}
		if b.CastlingRights&CastleWhiteQueen != 0 && HasBit(b.Colors[White]&b.Pieces[Rook], 0) && HasBit(b.Colors[White]&b.Pieces[King], 4) {
			if !HasBit(occupied, 1) && !HasBit(occupied, 2) && !HasBit(occupied, 3) {
				if !isSquareAttacked(b, 4, enemy) && !isSquareAttacked(b, 3, enemy) && !isSquareAttacked(b, 2, enemy) {
					moves = append(moves, Move{From: 4, To: 2, Flag: FlagCastleQueen})
				}
			}
		}
	} else {
		if b.CastlingRights&CastleBlackKing != 0 && HasBit(b.Colors[Black]&b.Pieces[Rook], 63) && HasBit(b.Colors[Black]&b.Pieces[King], 60) {
			if !HasBit(occupied, 61) && !HasBit(occupied, 62) {
				if !isSquareAttacked(b, 60, enemy) && !isSquareAttacked(b, 61, enemy) && !isSquareAttacked(b, 62, enemy) {
					moves = append(moves, Move{From: 60, To: 62, Flag: FlagCastleKing})
				}
			}
		}
		if b.CastlingRights&CastleBlackQueen != 0 && HasBit(b.Colors[Black]&b.Pieces[Rook], 56) && HasBit(b.Colors[Black]&b.Pieces[King], 60) {
			if !HasBit(occupied, 57) && !HasBit(occupied, 58) && !HasBit(occupied, 59) {
				if !isSquareAttacked(b, 60, enemy) && !isSquareAttacked(b, 59, enemy) && !isSquareAttacked(b, 58, enemy) {
					moves = append(moves, Move{From: 60, To: 58, Flag: FlagCastleQueen})
				}
			}
		}
	}
	return moves
}

// GenerateLegalMoves returns only legal moves
// (filtering out those that leave the king in check).
func GenerateLegalMoves(b *Board) []Move {
	pseudos := GeneratePseudoLegalMoves(b)
	legal := make([]Move, 0, len(pseudos))
	for _, m := range pseudos {
		nb := MakeMove(b, m)
		if !IsInCheck(nb, b.Turn) {
			legal = append(legal, m)
		}
	}
	return legal
}

// MakeMove returns a new board with the move applied
func MakeMove(b *Board, m Move) *Board {
	nb := b.Copy()

	nb.Hash ^= ZobristCastling[b.CastlingRights]
	if b.hasLegalEnPassantCapture() {
		nb.Hash ^= ZobristEnPassant[b.EnPassantSquare%8]
	}

	pieceColor := b.Turn
	var pieceType PieceType
	for pt := Pawn; pt <= King; pt++ {
		if HasBit(b.Pieces[pt], m.From) {
			pieceType = pt
			break
		}
	}

	var capturedType PieceType
	if HasBit(b.Colors[pieceColor.Other()], m.To) {
		for pt := Pawn; pt <= King; pt++ {
			if HasBit(b.Pieces[pt], m.To) {
				capturedType = pt
				break
			}
		}
	}

	// piece from
	ClearBit(&nb.Colors[pieceColor], m.From)
	ClearBit(&nb.Pieces[pieceType], m.From)
	nb.Hash ^= ZobristPieces[pieceColor][pieceType][m.From]

	// captured piece
	if capturedType != NoPieceType {
		ClearBit(&nb.Colors[pieceColor.Other()], m.To)
		ClearBit(&nb.Pieces[capturedType], m.To)
		nb.Hash ^= ZobristPieces[pieceColor.Other()][capturedType][m.To]
	}

	// piece to
	SetBit(&nb.Colors[pieceColor], m.To)
	SetBit(&nb.Pieces[pieceType], m.To)
	nb.Hash ^= ZobristPieces[pieceColor][pieceType][m.To]

	// En Passant
	if m.Flag == FlagEnPassant {
		captureSq := SquareIndex(FileOf(m.To), RankOf(m.From))
		ClearBit(&nb.Colors[pieceColor.Other()], captureSq)
		ClearBit(&nb.Pieces[Pawn], captureSq)
		nb.Hash ^= ZobristPieces[pieceColor.Other()][Pawn][captureSq]
	}

	// promotion
	if m.Flag == FlagPromotion {
		ClearBit(&nb.Pieces[Pawn], m.To)
		SetBit(&nb.Pieces[m.Promotion], m.To)
		nb.Hash ^= ZobristPieces[pieceColor][Pawn][m.From]      //remove promoting pawn
		nb.Hash ^= ZobristPieces[pieceColor][m.Promotion][m.To] //add promotion piece
	}

	// castling
	if m.Flag == FlagCastleKing {
		rank := RankOf(m.From)
		hRook, fRook := SquareIndex(7, rank), SquareIndex(5, rank)
		ClearBit(&nb.Colors[pieceColor], hRook)
		ClearBit(&nb.Pieces[Rook], hRook)
		SetBit(&nb.Colors[pieceColor], fRook)
		SetBit(&nb.Pieces[Rook], fRook)
		nb.Hash ^= ZobristPieces[pieceColor][Rook][hRook] //remove ROOk
		nb.Hash ^= ZobristPieces[pieceColor][Rook][fRook] //add rook
	}
	if m.Flag == FlagCastleQueen {
		rank := RankOf(m.From)
		aRook, dRook := SquareIndex(0, rank), SquareIndex(3, rank)
		ClearBit(&nb.Colors[pieceColor], aRook)
		ClearBit(&nb.Pieces[Rook], aRook)
		SetBit(&nb.Colors[pieceColor], dRook)
		SetBit(&nb.Pieces[Rook], dRook)
		nb.Hash ^= ZobristPieces[pieceColor][Rook][aRook]
		nb.Hash ^= ZobristPieces[pieceColor][Rook][dRook]
	}

	// update en passant square
	nb.EnPassantSquare = -1
	if m.Flag == FlagDoublePush {
		epRank := (RankOf(m.From) + RankOf(m.To)) / 2
		nb.EnPassantSquare = int8(SquareIndex(FileOf(m.From), epRank))
	}

	// update castling rights
	if pieceType == King {
		if pieceColor == White {
			nb.CastlingRights &^= CastleWhiteKing | CastleWhiteQueen
		} else {
			nb.CastlingRights &^= CastleBlackKing | CastleBlackQueen
		}
	}
	updateCastlingForRook := func(sq int) {
		switch sq {
		case SquareIndex(0, 0):
			nb.CastlingRights &^= CastleWhiteQueen
		case SquareIndex(7, 0):
			nb.CastlingRights &^= CastleWhiteKing
		case SquareIndex(0, 7):
			nb.CastlingRights &^= CastleBlackQueen
		case SquareIndex(7, 7):
			nb.CastlingRights &^= CastleBlackKing
		}
	}
	if pieceType == Rook {
		updateCastlingForRook(m.From)
	}
	if capturedType == Rook {
		updateCastlingForRook(m.To)
	}

	// update clocks
	if pieceType == Pawn || capturedType != NoPieceType || m.Flag == FlagEnPassant {
		nb.HalfMoveClock = 0
		nb.HistoryLength = 0
	} else {
		nb.HalfMoveClock++
		nb.History[nb.HistoryLength] = b.Hash
		nb.HistoryLength++
	}
	if b.Turn == Black {
		nb.FullMoveNumber++
	}

	nb.Turn = b.Turn.Other()
	nb.Hash ^= ZobristTurn

	nb.Hash ^= ZobristCastling[nb.CastlingRights]
	if nb.hasLegalEnPassantCapture() {
		nb.Hash ^= ZobristEnPassant[nb.EnPassantSquare%8]
	}

	return nb
}

// Game outcome helpers
type Outcome int

const (
	NoOutcome Outcome = iota
	WhiteWins
	BlackWins
	Draw
)

func (o Outcome) String() string {
	switch o {
	case WhiteWins:
		return "1-0"
	case BlackWins:
		return "0-1"
	case Draw:
		return "1/2-1/2"
	default:
		return "*"
	}
}

type Method int

const (
	NoMethod Method = iota
	Checkmate
	Stalemate
	FiftyMoveRule
	InsufficientMaterial
	ThreefoldRepetition
)

func (m Method) String() string {
	switch m {
	case Checkmate:
		return "Checkmate"
	case Stalemate:
		return "Stalemate"
	case FiftyMoveRule:
		return "Fifty-move rule"
	case InsufficientMaterial:
		return "Insufficient material"
	case ThreefoldRepetition:
		return "Threefold repetition"
	default:
		return ""
	}
}

func GetOutcome(b *Board) (Outcome, Method) {
	legalMoves := GenerateLegalMoves(b)

	if len(legalMoves) == 0 {
		if IsInCheck(b, b.Turn) {
			if b.Turn == White {
				return BlackWins, Checkmate
			}
			return WhiteWins, Checkmate
		}
		return Draw, Stalemate
	}

	if b.HalfMoveClock >= 100 {
		return Draw, FiftyMoveRule
	}

	if isRepetition(b) {
		return Draw, ThreefoldRepetition
	}

	if isInsufficientMaterial(b) {
		return Draw, InsufficientMaterial
	}

	return NoOutcome, NoMethod
}

func isInsufficientMaterial(b *Board) bool {
	whiteKnights := bits.OnesCount64(b.Pieces[Knight] & b.Colors[White])
	whiteBishops := bits.OnesCount64(b.Pieces[Bishop] & b.Colors[White])
	blackKnights := bits.OnesCount64(b.Pieces[Knight] & b.Colors[Black])
	blackBishops := bits.OnesCount64(b.Pieces[Bishop] & b.Colors[Black])

	whitePieces := bits.OnesCount64(b.Colors[White] &^ b.Pieces[King])
	blackPieces := bits.OnesCount64(b.Colors[Black] &^ b.Pieces[King])

	totalPieces := whitePieces + blackPieces

	if totalPieces == 0 {
		return true
	}
	if totalPieces == 1 {
		if whiteKnights == 1 || whiteBishops == 1 || blackKnights == 1 || blackBishops == 1 {
			return true
		}
	}

	return false
}

func isRepetition(b *Board) bool {
	repetitionCount := 1

	for i := b.HistoryLength - 2; i >= 0; i -= 2 {
		if b.History[i] == b.Hash {
			repetitionCount++
			if repetitionCount >= 3 {
				return true
			}
		}
	}

	return false
}
