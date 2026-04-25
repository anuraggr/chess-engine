package main

import (
	"encoding/binary"
	"math/rand"
	"os"
)

type BookEntry struct {
	Key    uint64
	Move   uint16
	Weight uint16
}

var OpeningBook []BookEntry
var BookLoaded bool

// load .bin file
func LoadBook(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	numEntries := len(data) / 16
	OpeningBook = make([]BookEntry, numEntries)

	for i := 0; i < numEntries; i++ {
		offset := i * 16
		OpeningBook[i].Key = binary.BigEndian.Uint64(data[offset : offset+8])
		OpeningBook[i].Move = binary.BigEndian.Uint16(data[offset+8 : offset+10])
		OpeningBook[i].Weight = binary.BigEndian.Uint16(data[offset+10 : offset+12])
	}

	BookLoaded = true
	return nil
}

func GetBookMove(b *Board) (Move, bool) {
	if !BookLoaded || len(OpeningBook) == 0 {
		return Move{}, false
	}

	key := b.Hash

	// lower bound BS
	left, right := 0, len(OpeningBook)
	for left < right {
		mid := int(uint(left+right) >> 1)
		if OpeningBook[mid].Key < key {
			left = mid + 1
		} else {
			right = mid
		}
	}

	if left >= len(OpeningBook) || OpeningBook[left].Key != key {
		return Move{}, false //pos not found in book
	}
	firstIndex := left

	var entries []BookEntry
	totalWeight := 0

	for i := firstIndex; i < len(OpeningBook) && OpeningBook[i].Key == key; i++ {
		if OpeningBook[i].Weight > 0 {
			entries = append(entries, OpeningBook[i])
			totalWeight += int(OpeningBook[i].Weight)
		}
	}

	if len(entries) == 0 {
		return Move{}, false
	}

	randWeight := rand.Intn(totalWeight)
	currentWeight := 0
	var chosenMove uint16

	for _, entry := range entries {
		currentWeight += int(entry.Weight)
		if currentWeight > randWeight {
			chosenMove = entry.Move
			break
		}
	}

	toFile := int(chosenMove & 7)
	toRank := int((chosenMove >> 3) & 7)
	fromFile := int((chosenMove >> 6) & 7)
	fromRank := int((chosenMove >> 9) & 7)

	fromSq := SquareIndex(fromFile, fromRank)
	toSq := SquareIndex(toFile, toRank)

	polyglotPromo := (chosenMove >> 12) & 7
	var enginePromo PieceType
	switch polyglotPromo {
	case 1:
		enginePromo = Knight
	case 2:
		enginePromo = Bishop
	case 3:
		enginePromo = Rook
	case 4:
		enginePromo = Queen
	default:
		enginePromo = NoPieceType
	}

	legalMoves := GenerateLegalMoves(b)
	for _, m := range legalMoves {
		if m.From == fromSq && m.To == toSq && m.Promotion == enginePromo {
			return m, true
		}
	}

	return Move{}, false
}
