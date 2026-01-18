package skiplist

import (
	"math/rand"
	"sync"
	"time"
)

const (
	maxLevel  = 16
	skipListP = 0.25
)

var (
	rnd  *rand.Rand
	once sync.Once
)

func initRand() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// SkipListNode represents a node in the skip list
type SkipListNode struct {
	Score   float64
	Member  string
	Forward []*SkipListNode
}

// SkipList implements the sorted set data structure
type SkipList struct {
	Header *SkipListNode
	Tail   *SkipListNode
	Length int64
	Level  int
}

// NewSkipList creates a new skip list
func NewSkipList() *SkipList {
	header := &SkipListNode{
		Forward: make([]*SkipListNode, maxLevel),
	}
	return &SkipList{
		Header: header,
		Level:  1,
		Length: 0,
	}
}

// randomLevel generates a random level for a new node
func randomLevel() int {
	once.Do(initRand)
	level := 1
	for rnd.Float64() < skipListP && level < maxLevel {
		level++
	}
	return level
}

// Insert adds or updates a node in the skip list
func (sl *SkipList) Insert(score float64, member string) *SkipListNode {
	update := make([]*SkipListNode, maxLevel)
	x := sl.Header
	for i := sl.Level - 1; i >= 0; i-- {
		for x.Forward[i] != nil && (x.Forward[i].Score < score || (x.Forward[i].Score == score && x.Forward[i].Member < member)) {
			x = x.Forward[i]
		}
		update[i] = x
	}
	x = x.Forward[0]

	if x != nil && x.Score == score && x.Member == member {
		return x
	}
	level := randomLevel()
	if level > sl.Level {
		for i := sl.Level; i < level; i++ {
			update[i] = sl.Header
		}
		sl.Level = level
	}
	newNode := &SkipListNode{
		Score:   score,
		Member:  member,
		Forward: make([]*SkipListNode, level),
	}
	for i := 0; i < level; i++ {
		newNode.Forward[i] = update[i].Forward[i]
		update[i].Forward[i] = newNode
	}
	if newNode.Forward[0] == nil {
		sl.Tail = newNode
	}
	sl.Length++
	return newNode
}

// Delete removes a node from the skip list
func (sl *SkipList) Delete(score float64, member string) bool {
	return true
}

// Search finds a node by score and member
func (sl *SkipList) Search(score float64, member string) *SkipListNode {
	return nil
}

// GetRank returns the rank (0-based) of a member
func (sl *SkipList) GetRank(score float64, member string) int64 {
	return 0
}

// GetByRank returns the node at the specified rank (0-based)
func (sl *SkipList) GetByRank(rank int64) *SkipListNode {
	return 0
}
