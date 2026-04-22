package main

const (
	FlagExact uint8 = 0 //Accurate score
	FlagAlpha uint8 = 1 //UpperBound. the score is atmost this
	FlagBeta  uint8 = 2 //the score is atleast this
)

type TTEntry struct {
	Key      uint64
	BestMove PackedMove
	Score    int16
	Depth    uint8
	Flag     uint8
}

type TranspositionTable struct {
	Entries []TTEntry
	Mask    uint64
}

func NewTT(sizeInMB int) *TranspositionTable {
	entrySizeByte := 16
	targetEnteris := (sizeInMB * 1024 * 1024) / entrySizeByte

	var numEntries uint64 = 1
	for numEntries <= uint64(targetEnteris) {
		numEntries *= 2
	}
	numEntries /= 2

	return &TranspositionTable{
		Entries: make([]TTEntry, numEntries),
		Mask:    numEntries - 1,
	}
}

var TT *TranspositionTable

func init() {
	TT = NewTT(16)
}

func (tt *TranspositionTable) Store(key uint64, depth uint8, score int16, flag uint8, bestMove Move) {
	index := key & tt.Mask
	entry := &tt.Entries[index]

	if entry.Key == 0 || entry.Key == key || depth >= entry.Depth {
		entry.Key = key
		entry.BestMove = bestMove.Pack() // compress the move
		entry.Score = score
		entry.Depth = depth
		entry.Flag = flag
	}
}

func (tt *TranspositionTable) Probe(key uint64) (TTEntry, bool) {
	index := key & tt.Mask
	entry := tt.Entries[index]

	if entry.Key == key {
		return entry, true // cache miss
	}

	return TTEntry{}, false // cache miss
}

func (tt *TranspositionTable) Clear() {
	for i := range tt.Entries {
		tt.Entries[i] = TTEntry{}
	}
}
