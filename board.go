package main

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

type Color uint8

const (
	White Color = iota
	Black
)

func (c Color) Other() Color {
	if c == White {
		return Black
	}
	return White
}

func (c Color) String() string {
	if c == White {
		return "w"
	}
	return "b"
}

type PieceType uint8

const (
	NoPieceType PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// bit 3 = color (0=white, 1=black), bits 0-2 = piece type.
type Piece uint8

const NoPiece Piece = 0

func NewPiece(pt PieceType, c Color) Piece {
	return Piece(uint8(c)<<3 | uint8(pt))
}

func (p Piece) Type() PieceType {
	//To read only the piece type, we have to remove color info.
	// AND operation with 0..111 will only leave the last 3 bits.
	return PieceType(p & 0x07)
}
func (p Piece) Color() Color {
	//right shift 3 bits to only be left with color information of a piece
	return Color((p >> 3) & 1)
}

var pieceStrs = [16]string{
	".", "P", "N", "B", "R", "Q", "K", "?",
	"?", "p", "n", "b", "r", "q", "k", "?",
}

func (p Piece) String() string {
	if p < 16 {
		return pieceStrs[p]
	}
	return "?"
}

// square helper funcs
func SquareIndex(file, rank int) int { return rank*8 + file }
func FileOf(sq int) int              { return sq % 8 }
func RankOf(sq int) int              { return sq / 8 }

func SquareFromAlgebraic(s string) (int, error) {
	if len(s) != 2 {
		return -1, fmt.Errorf("invalid square: %s", s)
	}
	file := int(s[0] - 'a')
	rank := int(s[1] - '1')
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1, fmt.Errorf("invalid square: %s", s)
	}
	return SquareIndex(file, rank), nil
}

func SquareToAlgebraic(sq int) string {
	return string(rune('a'+FileOf(sq))) + string(rune('1'+RankOf(sq)))
}

const (
	//We can use a single int to represent entire castling rights at any point
	CastleWhiteKing  uint8 = 1 << iota // K
	CastleWhiteQueen                   // Q
	CastleBlackKing                    // k
	CastleBlackQueen                   // q
)

// board
type Board struct {
	Colors          [2]uint64
	Pieces          [7]uint64
	Turn            Color
	CastlingRights  uint8
	EnPassantSquare int8 // -1 if none
	HalfMoveClock   int
	FullMoveNumber  int
	Hash            uint64
	History         []uint64
}

// bit manipulation helpers
func SetBit(bb *uint64, sq int) {
	*bb |= 1 << sq
}

func ClearBit(bb *uint64, sq int) {
	*bb &= ^(1 << sq)
}

func PopBit(bb *uint64) int {
	sq := bits.TrailingZeros64(*bb)
	*bb &= *bb - 1
	return sq
}

func HasBit(bb uint64, sq int) bool {
	return (bb & (1 << sq)) != 0
}

func (b *Board) AddPiece(sq int, p Piece) {
	if p == NoPiece {
		return
	}
	SetBit(&b.Colors[p.Color()], sq)
	SetBit(&b.Pieces[p.Type()], sq)
}

func (b *Board) RemovePiece(sq int, p Piece) {
	if p == NoPiece {
		return
	}
	ClearBit(&b.Colors[p.Color()], sq)
	ClearBit(&b.Pieces[p.Type()], sq)
}

func (b *Board) PieceAt(sq int) Piece {
	bit := uint64(1) << sq

	color := White
	if (b.Colors[White] & bit) == 0 {
		if (b.Colors[Black] & bit) == 0 {
			return NoPiece
		}
		color = Black
	}

	if (b.Pieces[Pawn] & bit) != 0 {
		return NewPiece(Pawn, color)
	}
	if (b.Pieces[Knight] & bit) != 0 {
		return NewPiece(Knight, color)
	}
	if (b.Pieces[Bishop] & bit) != 0 {
		return NewPiece(Bishop, color)
	}
	if (b.Pieces[Rook] & bit) != 0 {
		return NewPiece(Rook, color)
	}
	if (b.Pieces[Queen] & bit) != 0 {
		return NewPiece(Queen, color)
	}
	if (b.Pieces[King] & bit) != 0 {
		return NewPiece(King, color)
	}

	return NoPiece
}

func NewBoard() *Board {
	b := &Board{
		Turn:            White,
		CastlingRights:  CastleWhiteKing | CastleWhiteQueen | CastleBlackKing | CastleBlackQueen,
		EnPassantSquare: -1,
		HalfMoveClock:   0,
		FullMoveNumber:  1,
	}

	// pawns
	for f := 0; f < 8; f++ {
		b.AddPiece(SquareIndex(f, 1), NewPiece(Pawn, White))
		b.AddPiece(SquareIndex(f, 6), NewPiece(Pawn, Black))
	}

	// pieces
	order := []PieceType{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}
	for f, pt := range order {
		b.AddPiece(SquareIndex(f, 0), NewPiece(pt, White))
		b.AddPiece(SquareIndex(f, 7), NewPiece(pt, Black))
	}

	b.ComputeHash()

	return b
}

// FEN
func BoardFromFEN(fen string) (*Board, error) {
	parts := strings.Fields(fen)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid FEN: need at least 4 fields, got %d", len(parts))
	}

	b := &Board{EnPassantSquare: -1, HalfMoveClock: 0, FullMoveNumber: 1}

	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("invalid FEN: need 8 ranks, got %d", len(ranks))
	}
	for r := 7; r >= 0; r-- {
		file := 0
		for _, ch := range ranks[7-r] {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				continue
			}
			pt, color, err := pieceFromChar(ch)
			if err != nil {
				return nil, err
			}
			b.AddPiece(SquareIndex(file, r), NewPiece(pt, color))
			file++
		}
	}

	switch parts[1] {
	case "w":
		b.Turn = White
	case "b":
		b.Turn = Black
	default:
		return nil, fmt.Errorf("invalid FEN active color: %s", parts[1])
	}

	b.CastlingRights = 0
	if parts[2] != "-" {
		for _, ch := range parts[2] {
			switch ch {
			case 'K':
				b.CastlingRights |= CastleWhiteKing
			case 'Q':
				b.CastlingRights |= CastleWhiteQueen
			case 'k':
				b.CastlingRights |= CastleBlackKing
			case 'q':
				b.CastlingRights |= CastleBlackQueen
			}
		}
	}

	// this is the en passant target square
	if parts[3] != "-" {
		sq, err := SquareFromAlgebraic(parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid FEN en passant: %v", err)
		}
		b.EnPassantSquare = int8(sq)
	}

	// 5. half-move clock (this can be optional)
	if len(parts) > 4 {
		hmc, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, fmt.Errorf("invalid FEN half-move clock: %v", err)
		}
		b.HalfMoveClock = hmc
	}

	// 6. full-move number (optional)
	if len(parts) > 5 {
		fmn, err := strconv.Atoi(parts[5])
		if err != nil {
			return nil, fmt.Errorf("invalid FEN full-move number: %v", err)
		}
		b.FullMoveNumber = fmn
	}

	b.ComputeHash()

	return b, nil
}

func pieceFromChar(ch rune) (PieceType, Color, error) {
	color := White
	if ch >= 'a' && ch <= 'z' {
		color = Black
		ch = ch - 32 // to upper
	}
	switch ch {
	case 'P':
		return Pawn, color, nil
	case 'N':
		return Knight, color, nil
	case 'B':
		return Bishop, color, nil
	case 'R':
		return Rook, color, nil
	case 'Q':
		return Queen, color, nil
	case 'K':
		return King, color, nil
	}
	return NoPieceType, White, fmt.Errorf("invalid piece char: %c", ch)
}

// returns current boards FEN
func (b *Board) FEN() string {
	var sb strings.Builder

	for r := 7; r >= 0; r-- {
		empty := 0
		for f := 0; f < 8; f++ {
			p := b.PieceAt(SquareIndex(f, r))
			if p == NoPiece {
				empty++
				continue
			}
			if empty > 0 {
				sb.WriteByte(byte('0' + empty))
				empty = 0
			}
			sb.WriteString(p.String())
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if r > 0 {
			sb.WriteByte('/')
		}
	}

	sb.WriteByte(' ')

	sb.WriteString(b.Turn.String())
	sb.WriteByte(' ')

	if b.CastlingRights == 0 {
		sb.WriteByte('-')
	} else {
		if b.CastlingRights&CastleWhiteKing != 0 {
			sb.WriteByte('K')
		}
		if b.CastlingRights&CastleWhiteQueen != 0 {
			sb.WriteByte('Q')
		}
		if b.CastlingRights&CastleBlackKing != 0 {
			sb.WriteByte('k')
		}
		if b.CastlingRights&CastleBlackQueen != 0 {
			sb.WriteByte('q')
		}
	}
	sb.WriteByte(' ')

	if b.EnPassantSquare < 0 {
		sb.WriteByte('-')
	} else {
		sb.WriteString(SquareToAlgebraic(int(b.EnPassantSquare)))
	}

	sb.WriteString(fmt.Sprintf(" %d %d", b.HalfMoveClock, b.FullMoveNumber))

	return sb.String()
}

// terminal display
func (b *Board) String() string {
	var sb strings.Builder
	sb.WriteString("  a b c d e f g h\n")
	for r := 7; r >= 0; r-- {
		sb.WriteByte(byte('1' + r))
		sb.WriteByte(' ')
		for f := 0; f < 8; f++ {
			sb.WriteString(b.PieceAt(SquareIndex(f, r)).String())
			sb.WriteByte(' ')
		}
		sb.WriteByte(byte('1' + r))
		sb.WriteByte('\n')
	}
	sb.WriteString("  a b c d e f g h\n")
	return sb.String()
}

func (b *Board) Copy() *Board {
	nb := *b

	if b.History != nil {
		nb.History = make([]uint64, len(b.History))
		copy(nb.History, b.History)
	}

	return &nb
}
