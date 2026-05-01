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

// pre-allocated slice buffer and length
type MoveList struct {
	Moves  [256]Move
	Scores [256]int
	Count  int
}

func (ml *MoveList) Add(m Move) {
	ml.Moves[ml.Count] = m
	ml.Count++
}

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

// Bits 0–5: From square (0–63)
// Bits 6–11: To square (0–63)
// Bits 12–15: Special (flag + promotion combined)
// TODO: Make Unpacked usage everywhere. This is onlyused in TT right now
type PackedMove uint16

const NoPackedMove PackedMove = 0

func (m Move) Pack() PackedMove {

	var special uint16
	switch m.Flag {
	case FlagDoublePush:
		special = 1
	case FlagEnPassant:
		special = 2
	case FlagCastleKing:
		special = 3
	case FlagCastleQueen:
		special = 4
	case FlagPromotion:
		switch m.Promotion {
		case Knight:
			special = 5
		case Bishop:
			special = 6
		case Rook:
			special = 7
		case Queen:
			special = 8
		}
	}
	return PackedMove(uint16(m.From) | uint16(m.To)<<6 | special<<12)
}

func (pm PackedMove) Unpack() Move {

	from := int(pm & 0x3F)
	to := int((pm >> 6) & 0x3F)
	special := (pm >> 12) & 0xF

	m := Move{From: from, To: to}
	switch special {
	case 1:
		m.Flag = FlagDoublePush
	case 2:
		m.Flag = FlagEnPassant
	case 3:
		m.Flag = FlagCastleKing
	case 4:
		m.Flag = FlagCastleQueen
	case 5:
		m.Flag = FlagPromotion
		m.Promotion = Knight
	case 6:
		m.Flag = FlagPromotion
		m.Promotion = Bishop
	case 7:
		m.Flag = FlagPromotion
		m.Promotion = Rook
	case 8:
		m.Flag = FlagPromotion
		m.Promotion = Queen
	}
	return m
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
// Generate then filter is std pattern because its very fast and only
// very little moves are illegal
func appendPromotions(ml *MoveList, from, to int) {
	for _, pt := range []PieceType{Queen, Rook, Bishop, Knight} {
		ml.Add(Move{From: from, To: to, Promotion: pt, Flag: FlagPromotion})
	}
}

func GeneratePseudoLegalMoves(b *Board, ml *MoveList) {
	ml.Count = 0
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
				appendPromotions(ml, from, to)
			} else {
				ml.Add(Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		// double push
		doublePushed := ((pushed & 0x0000000000FF0000) << 8) & empty
		bb = doublePushed
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to - 16
			ml.Add(Move{From: from, To: to, Flag: FlagDoublePush})
			bb &= bb - 1
		}
		// captures left
		capsL := (pawns << 7) & 0x7F7F7F7F7F7F7F7F & b.Colors[Black]
		bb = capsL
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to - 7
			if to >= 56 {
				appendPromotions(ml, from, to)
			} else {
				ml.Add(Move{From: from, To: to})
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
				appendPromotions(ml, from, to)
			} else {
				ml.Add(Move{From: from, To: to})
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
				ml.Add(Move{From: from, To: epSq, Flag: FlagEnPassant})
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
				appendPromotions(ml, from, to)
			} else {
				ml.Add(Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		doublePushed := ((pushed & 0x0000FF0000000000) >> 8) & empty
		bb = doublePushed
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 16
			ml.Add(Move{From: from, To: to, Flag: FlagDoublePush})
			bb &= bb - 1
		}
		capsR := (pawns >> 7) & 0xFEFEFEFEFEFEFEFE & b.Colors[White]
		bb = capsR
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 7
			if to <= 7 {
				appendPromotions(ml, from, to)
			} else {
				ml.Add(Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		capsL := (pawns >> 9) & 0x7F7F7F7F7F7F7F7F & b.Colors[White]
		bb = capsL
		for bb != 0 {
			to := bits.TrailingZeros64(bb)
			from := to + 9
			if to <= 7 {
				appendPromotions(ml, from, to)
			} else {
				ml.Add(Move{From: from, To: to})
			}
			bb &= bb - 1
		}
		if b.EnPassantSquare >= 0 {
			epSq := int(b.EnPassantSquare)
			caps := PawnAttacks[White][epSq] & pawns
			bb = caps
			for bb != 0 {
				from := bits.TrailingZeros64(bb)
				ml.Add(Move{From: from, To: epSq, Flag: FlagEnPassant})
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
			ml.Add(Move{From: from, To: to})
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
			ml.Add(Move{From: from, To: to})
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
			ml.Add(Move{From: from, To: to})
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
			ml.Add(Move{From: from, To: to})
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
			ml.Add(Move{From: from, To: to})
			attacks &= attacks - 1
		}
	}

	generateCastlingMoves(b, ml)
}

func generateCastlingMoves(b *Board, ml *MoveList) {
	color := b.Turn
	enemy := color.Other()
	occupied := b.Colors[White] | b.Colors[Black]

	if color == White {
		if b.CastlingRights&CastleWhiteKing != 0 && HasBit(b.Colors[White]&b.Pieces[Rook], 7) && HasBit(b.Colors[White]&b.Pieces[King], 4) {
			if !HasBit(occupied, 5) && !HasBit(occupied, 6) {
				if !isSquareAttacked(b, 4, enemy) && !isSquareAttacked(b, 5, enemy) && !isSquareAttacked(b, 6, enemy) {
					ml.Add(Move{From: 4, To: 6, Flag: FlagCastleKing})
				}
			}
		}
		if b.CastlingRights&CastleWhiteQueen != 0 && HasBit(b.Colors[White]&b.Pieces[Rook], 0) && HasBit(b.Colors[White]&b.Pieces[King], 4) {
			if !HasBit(occupied, 1) && !HasBit(occupied, 2) && !HasBit(occupied, 3) {
				if !isSquareAttacked(b, 4, enemy) && !isSquareAttacked(b, 3, enemy) && !isSquareAttacked(b, 2, enemy) {
					ml.Add(Move{From: 4, To: 2, Flag: FlagCastleQueen})
				}
			}
		}
	} else {
		if b.CastlingRights&CastleBlackKing != 0 && HasBit(b.Colors[Black]&b.Pieces[Rook], 63) && HasBit(b.Colors[Black]&b.Pieces[King], 60) {
			if !HasBit(occupied, 61) && !HasBit(occupied, 62) {
				if !isSquareAttacked(b, 60, enemy) && !isSquareAttacked(b, 61, enemy) && !isSquareAttacked(b, 62, enemy) {
					ml.Add(Move{From: 60, To: 62, Flag: FlagCastleKing})
				}
			}
		}
		if b.CastlingRights&CastleBlackQueen != 0 && HasBit(b.Colors[Black]&b.Pieces[Rook], 56) && HasBit(b.Colors[Black]&b.Pieces[King], 60) {
			if !HasBit(occupied, 57) && !HasBit(occupied, 58) && !HasBit(occupied, 59) {
				if !isSquareAttacked(b, 60, enemy) && !isSquareAttacked(b, 59, enemy) && !isSquareAttacked(b, 58, enemy) {
					ml.Add(Move{From: 60, To: 58, Flag: FlagCastleQueen})
				}
			}
		}
	}
}

func GenerateLegalMoves(b *Board) []Move {
	var ml MoveList
	GeneratePseudoLegalMoves(b, &ml)
	legal := make([]Move, 0, ml.Count)
	for i := 0; i < ml.Count; i++ {
		m := ml.Moves[i]
		info := MakeMove(b, m)
		if !IsInCheck(b, b.Turn.Other()) {
			legal = append(legal, m)
		}
		UnmakeMove(b, m, info)
	}
	return legal
}

type UndoInfo struct {
	Captured        PieceType
	CastlingRights  uint8
	EnPassantSquare int8
	HalfMoveClock   int
	Hash            uint64
}

func MakeMove(b *Board, m Move) UndoInfo {
	info := UndoInfo{
		CastlingRights:  b.CastlingRights,
		EnPassantSquare: b.EnPassantSquare,
		HalfMoveClock:   b.HalfMoveClock,
		Hash:            b.Hash,
		Captured:        NoPieceType,
	}

	b.Hash ^= ZobristCastling[b.CastlingRights]
	if b.hasLegalEnPassantCapture() {
		b.Hash ^= ZobristEnPassant[b.EnPassantSquare%8]
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
		info.Captured = capturedType
	}

	// piece from
	ClearBit(&b.Colors[pieceColor], m.From)
	ClearBit(&b.Pieces[pieceType], m.From)
	b.Hash ^= ZobristPieces[pieceColor][pieceType][m.From]

	// captured piece
	if capturedType != NoPieceType {
		ClearBit(&b.Colors[pieceColor.Other()], m.To)
		ClearBit(&b.Pieces[capturedType], m.To)
		b.Hash ^= ZobristPieces[pieceColor.Other()][capturedType][m.To]
		if capturedType >= Knight && capturedType <= Queen {
			b.TotalMaterial -= PieceValue[capturedType]
		}
	}

	// piece to
	SetBit(&b.Colors[pieceColor], m.To)
	SetBit(&b.Pieces[pieceType], m.To)
	b.Hash ^= ZobristPieces[pieceColor][pieceType][m.To]

	// En Passant
	if m.Flag == FlagEnPassant {
		captureSq := SquareIndex(FileOf(m.To), RankOf(m.From))
		ClearBit(&b.Colors[pieceColor.Other()], captureSq)
		ClearBit(&b.Pieces[Pawn], captureSq)
		b.Hash ^= ZobristPieces[pieceColor.Other()][Pawn][captureSq]
	}

	// promotion
	if m.Flag == FlagPromotion {
		ClearBit(&b.Pieces[Pawn], m.To)
		SetBit(&b.Pieces[m.Promotion], m.To)
		b.Hash ^= ZobristPieces[pieceColor][Pawn][m.To]        //remove promoting pawn
		b.Hash ^= ZobristPieces[pieceColor][m.Promotion][m.To] //add promotion piece
		b.TotalMaterial += PieceValue[m.Promotion]
	}

	// castling
	if m.Flag == FlagCastleKing {
		rank := RankOf(m.From)
		hRook, fRook := SquareIndex(7, rank), SquareIndex(5, rank)
		ClearBit(&b.Colors[pieceColor], hRook)
		ClearBit(&b.Pieces[Rook], hRook)
		SetBit(&b.Colors[pieceColor], fRook)
		SetBit(&b.Pieces[Rook], fRook)
		b.Hash ^= ZobristPieces[pieceColor][Rook][hRook] //remove ROOk
		b.Hash ^= ZobristPieces[pieceColor][Rook][fRook] //add rook
	}
	if m.Flag == FlagCastleQueen {
		rank := RankOf(m.From)
		aRook, dRook := SquareIndex(0, rank), SquareIndex(3, rank)
		ClearBit(&b.Colors[pieceColor], aRook)
		ClearBit(&b.Pieces[Rook], aRook)
		SetBit(&b.Colors[pieceColor], dRook)
		SetBit(&b.Pieces[Rook], dRook)
		b.Hash ^= ZobristPieces[pieceColor][Rook][aRook]
		b.Hash ^= ZobristPieces[pieceColor][Rook][dRook]
	}

	// update en passant square
	b.EnPassantSquare = -1
	if m.Flag == FlagDoublePush {
		epRank := (RankOf(m.From) + RankOf(m.To)) / 2
		b.EnPassantSquare = int8(SquareIndex(FileOf(m.From), epRank))
	}

	// update castling rights
	if pieceType == King {
		if pieceColor == White {
			b.CastlingRights &^= CastleWhiteKing | CastleWhiteQueen
		} else {
			b.CastlingRights &^= CastleBlackKing | CastleBlackQueen
		}
	}
	updateCastlingForRook := func(sq int) {
		switch sq {
		case SquareIndex(0, 0):
			b.CastlingRights &^= CastleWhiteQueen
		case SquareIndex(7, 0):
			b.CastlingRights &^= CastleWhiteKing
		case SquareIndex(0, 7):
			b.CastlingRights &^= CastleBlackQueen
		case SquareIndex(7, 7):
			b.CastlingRights &^= CastleBlackKing
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
		b.HalfMoveClock = 0
	} else {
		b.HalfMoveClock++
	}
	b.History[b.HistoryLength] = info.Hash
	b.HistoryLength++

	if b.Turn == Black {
		b.FullMoveNumber++
	}

	b.Turn = b.Turn.Other()
	b.Hash ^= ZobristTurn

	b.Hash ^= ZobristCastling[b.CastlingRights]
	if b.hasLegalEnPassantCapture() {
		b.Hash ^= ZobristEnPassant[b.EnPassantSquare%8]
	}

	return info
}

func UnmakeMove(b *Board, m Move, info UndoInfo) {
	b.Turn = b.Turn.Other()
	pieceColor := b.Turn

	var pieceType PieceType
	if m.Flag == FlagPromotion {
		pieceType = Pawn
	} else {
		for pt := Pawn; pt <= King; pt++ {
			if HasBit(b.Pieces[pt], m.To) {
				pieceType = pt
				break
			}
		}
	}

	if m.Flag == FlagPromotion {
		ClearBit(&b.Colors[pieceColor], m.To)
		ClearBit(&b.Pieces[m.Promotion], m.To)
		SetBit(&b.Colors[pieceColor], m.From)
		SetBit(&b.Pieces[Pawn], m.From)
		b.TotalMaterial -= PieceValue[m.Promotion]
	} else {
		ClearBit(&b.Colors[pieceColor], m.To)
		ClearBit(&b.Pieces[pieceType], m.To)
		SetBit(&b.Colors[pieceColor], m.From)
		SetBit(&b.Pieces[pieceType], m.From)
	}

	if info.Captured != NoPieceType {
		SetBit(&b.Colors[pieceColor.Other()], m.To)
		SetBit(&b.Pieces[info.Captured], m.To)
		if info.Captured >= Knight && info.Captured <= Queen {
			b.TotalMaterial += PieceValue[info.Captured]
		}
	}

	if m.Flag == FlagEnPassant {
		captureSq := SquareIndex(FileOf(m.To), RankOf(m.From))
		SetBit(&b.Colors[pieceColor.Other()], captureSq)
		SetBit(&b.Pieces[Pawn], captureSq)
	}

	if m.Flag == FlagCastleKing {
		rank := RankOf(m.From)
		hRook, fRook := SquareIndex(7, rank), SquareIndex(5, rank)
		ClearBit(&b.Colors[pieceColor], fRook)
		ClearBit(&b.Pieces[Rook], fRook)
		SetBit(&b.Colors[pieceColor], hRook)
		SetBit(&b.Pieces[Rook], hRook)
	} else if m.Flag == FlagCastleQueen {
		rank := RankOf(m.From)
		aRook, dRook := SquareIndex(0, rank), SquareIndex(3, rank)
		ClearBit(&b.Colors[pieceColor], dRook)
		ClearBit(&b.Pieces[Rook], dRook)
		SetBit(&b.Colors[pieceColor], aRook)
		SetBit(&b.Pieces[Rook], aRook)
	}

	b.CastlingRights = info.CastlingRights
	b.EnPassantSquare = info.EnPassantSquare
	b.HalfMoveClock = info.HalfMoveClock
	b.Hash = info.Hash

	if b.Turn == Black {
		b.FullMoveNumber--
	}
	b.HistoryLength--
}

func MakeNullMove(b *Board) UndoInfo {
	info := UndoInfo{
		CastlingRights:  b.CastlingRights,
		EnPassantSquare: b.EnPassantSquare,
		HalfMoveClock:   b.HalfMoveClock,
		Hash:            b.Hash,
		Captured:        NoPieceType,
	}

	if b.Turn == White {
		b.Turn = Black
	} else {
		b.Turn = White
	}

	b.Hash ^= ZobristTurn

	if b.EnPassantSquare != -1 {
		epFile := b.EnPassantSquare % 8
		b.Hash ^= ZobristEnPassant[epFile]
		b.EnPassantSquare = -1
	}

	b.History[b.HistoryLength] = info.Hash
	b.HistoryLength++
	b.HalfMoveClock++

	return info
}

func UnmakeNullMove(b *Board, info UndoInfo) {
	b.Turn = b.Turn.Other()
	b.EnPassantSquare = info.EnPassantSquare
	b.CastlingRights = info.CastlingRights
	b.HalfMoveClock = info.HalfMoveClock
	b.Hash = info.Hash
	b.HistoryLength--
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

	if isThreefoldRepetition(b) {
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

// isRepetitionForSearch returns true if the current position has appeared
// at least once before. Used inside negamax so the engine treats any
// repetition as a draw and avoids shuffling into threefold when winning.
func isRepetitionForSearch(b *Board) bool {
	limit := b.HistoryLength - b.HalfMoveClock
	if limit < 0 {
		limit = 0
	}

	for i := b.HistoryLength - 2; i >= limit; i -= 2 {
		if b.History[i] == b.Hash {
			return true
		}
	}

	return false
}

// isThreefoldRepetition requires 3 occurrences. Used only for
// official game-over adjudication in GetOutcome().
func isThreefoldRepetition(b *Board) bool {
	repetitionCount := 1

	limit := b.HistoryLength - b.HalfMoveClock
	if limit < 0 {
		limit = 0
	}

	for i := b.HistoryLength - 2; i >= limit; i -= 2 {
		if b.History[i] == b.Hash {
			repetitionCount++
			if repetitionCount >= 3 {
				return true
			}
		}
	}

	return false
}
