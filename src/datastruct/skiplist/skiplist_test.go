package skiplist

import (
	"testing"
)

func TestSkipList_Insert_EmptyList(t *testing.T) {
	sl := NewSkipList()

	// Test insert into empty list
	node := sl.Insert(10.0, "first")

	if node == nil {
		t.Fatal("Insert should return non-nil node")
	}
	if node.Score != 10.0 {
		t.Errorf("Expected score 10.0, got %f", node.Score)
	}
	if node.Member != "first" {
		t.Errorf("Expected member 'first', got '%s'", node.Member)
	}
	if sl.Length != 1 {
		t.Errorf("Expected length 1, got %d", sl.Length)
	}
	if sl.Tail != node {
		t.Error("Tail should point to the inserted node")
	}
}

func TestSkipList_Insert_MultipleNodes(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes in random order
	testData := []struct {
		score  float64
		member string
	}{
		{20.0, "second"},
		{10.0, "first"},
		{30.0, "third"},
		{15.0, "middle"},
	}

	for _, data := range testData {
		node := sl.Insert(data.score, data.member)
		if node == nil {
			t.Fatalf("Insert failed for score %f, member %s", data.score, data.member)
		}
		if node.Score != data.score {
			t.Errorf("Expected score %f, got %f", data.score, node.Score)
		}
		if node.Member != data.member {
			t.Errorf("Expected member %s, got %s", data.member, node.Member)
		}
	}

	if sl.Length != 4 {
		t.Errorf("Expected length 4, got %d", sl.Length)
	}
}

func TestSkipList_Insert_Ordering(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes
	sl.Insert(30.0, "third")
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(25.0, "middle")

	// Verify ordering by traversing level 0
	expected := []struct {
		score  float64
		member string
	}{
		{10.0, "first"},
		{20.0, "second"},
		{25.0, "middle"},
		{30.0, "third"},
	}

	current := sl.Header.Forward[0]
	for i, exp := range expected {
		if current == nil {
			t.Fatalf("Node %d is nil", i)
		}
		if current.Score != exp.score {
			t.Errorf("Node %d: expected score %f, got %f", i, exp.score, current.Score)
		}
		if current.Member != exp.member {
			t.Errorf("Node %d: expected member %s, got %s", i, exp.member, current.Member)
		}
		current = current.Forward[0]
	}

	if current != nil {
		t.Error("There should be no more nodes after the expected ones")
	}
}

func TestSkipList_Insert_SameScore_DifferentMembers(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes with same score, different members
	sl.Insert(10.0, "zebra")
	sl.Insert(10.0, "apple")
	sl.Insert(10.0, "banana")

	// Should be ordered lexicographically by member when scores are equal
	expected := []string{"apple", "banana", "zebra"}

	current := sl.Header.Forward[0]
	for i, expectedMember := range expected {
		if current == nil {
			t.Fatalf("Node %d is nil", i)
		}
		if current.Score != 10.0 {
			t.Errorf("Node %d: expected score 10.0, got %f", i, current.Score)
		}
		if current.Member != expectedMember {
			t.Errorf("Node %d: expected member %s, got %s", i, expectedMember, current.Member)
		}
		current = current.Forward[0]
	}
}

func TestSkipList_Insert_DuplicateNode(t *testing.T) {
	sl := NewSkipList()

	// Insert a node
	node1 := sl.Insert(10.0, "test")
	originalLength := sl.Length

	// Try to insert the same node again
	node2 := sl.Insert(10.0, "test")

	// Should return the existing node, not create a new one
	if node1 != node2 {
		t.Error("Insert should return the existing node for duplicate")
	}
	if sl.Length != originalLength {
		t.Errorf("Length should not change for duplicate insert. Expected %d, got %d", originalLength, sl.Length)
	}
}

func TestSkipList_Insert_TailPointerUpdate(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes in ascending order
	node1 := sl.Insert(10.0, "first")
	if sl.Tail != node1 {
		t.Error("Tail should point to first node")
	}

	node2 := sl.Insert(20.0, "second")
	if sl.Tail != node2 {
		t.Error("Tail should point to second node (highest)")
	}

	node3 := sl.Insert(30.0, "third")
	if sl.Tail != node3 {
		t.Error("Tail should point to third node (highest)")
	}

	// Insert a node in the middle - tail should not change
	sl.Insert(15.0, "middle")
	if sl.Tail != node3 {
		t.Error("Tail should still point to third node")
	}
}

func TestSkipList_Insert_LevelExpansion(t *testing.T) {
	sl := NewSkipList()

	if sl.Level != 1 {
		t.Errorf("Initial level should be 1, got %d", sl.Level)
	}

	// Insert multiple nodes to potentially trigger level expansion
	for i := 0; i < 20; i++ {
		sl.Insert(float64(i), "node")
	}

	// Verify all nodes are still reachable from level 0
	count := 0
	current := sl.Header.Forward[0]
	for current != nil {
		count++
		current = current.Forward[0]
	}

	if count != 20 {
		t.Errorf("Expected 20 nodes reachable from level 0, got %d", count)
	}

	if sl.Length != 20 {
		t.Errorf("Expected length 20, got %d", sl.Length)
	}
}

func TestSkipList_Insert_ForwardPointersConsistency(t *testing.T) {
	sl := NewSkipList()

	// Insert some nodes
	nodes := []struct {
		score  float64
		member string
	}{
		{10.0, "a"},
		{20.0, "b"},
		{30.0, "c"},
		{40.0, "d"},
		{50.0, "e"},
	}

	for _, n := range nodes {
		sl.Insert(n.score, n.member)
	}

	// Verify forward pointers consistency across all levels
	for level := 0; level < sl.Level; level++ {
		current := sl.Header
		var prevScore float64 = -1

		for current.Forward[level] != nil {
			current = current.Forward[level]

			// Verify scores are in ascending order
			if current.Score < prevScore {
				t.Errorf("Level %d: scores not in ascending order. Prev: %f, Current: %f",
					level, prevScore, current.Score)
			}
			prevScore = current.Score
		}
	}
}

func TestSkipList_Insert_StressTest(t *testing.T) {
	sl := NewSkipList()

	// Insert a large number of nodes
	numNodes := 100
	for i := 0; i < numNodes; i++ {
		// Insert in random order
		score := float64(i)
		if i%2 == 0 {
			score = float64(numNodes - i)
		}

		member := "node"
		sl.Insert(score, member)
	}

	// Verify final structure
	if sl.Length < 1 {
		t.Errorf("Expected length > 0, got %d", sl.Length)
	}

	// Verify ordering (traverse level 0)
	current := sl.Header.Forward[0]
	prevScore := float64(-1)
	count := 0

	for current != nil {
		count++
		if current.Score < prevScore {
			t.Errorf("Ordering violated at node %d. Prev: %f, Current: %f",
				count, prevScore, current.Score)
		}
		prevScore = current.Score
		current = current.Forward[0]
	}
}

// Benchmark tests
func BenchmarkSkipList_Insert(b *testing.B) {
	sl := NewSkipList()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Insert(float64(i), "node")
	}
}

func BenchmarkSkipList_Insert_Random(b *testing.B) {
	sl := NewSkipList()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		score := float64(i * 7 % 1000) // Pseudo-random pattern
		sl.Insert(score, "node")
	}
}

// ===== DELETE TESTS =====

func TestSkipList_Delete_EmptyList(t *testing.T) {
	sl := NewSkipList()

	// Try to delete from empty list
	result := sl.Delete(10.0, "nonexistent")

	if result {
		t.Error("Delete should return false for empty list")
	}
	if sl.Length != 0 {
		t.Errorf("Length should remain 0, got %d", sl.Length)
	}
}

func TestSkipList_Delete_NonExistentNode(t *testing.T) {
	sl := NewSkipList()

	// Insert some nodes
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(30.0, "third")

	// Try to delete non-existent nodes
	testCases := []struct {
		score  float64
		member string
		desc   string
	}{
		{5.0, "nonexistent", "score too low"},
		{35.0, "nonexistent", "score too high"},
		{15.0, "nonexistent", "score in middle but member doesn't exist"},
		{20.0, "wrong", "correct score but wrong member"},
	}

	originalLength := sl.Length

	for _, tc := range testCases {
		result := sl.Delete(tc.score, tc.member)
		if result {
			t.Errorf("Delete should return false for %s", tc.desc)
		}
		if sl.Length != originalLength {
			t.Errorf("Length should not change for %s. Expected %d, got %d", tc.desc, originalLength, sl.Length)
		}
	}
}

func TestSkipList_Delete_SingleNode(t *testing.T) {
	sl := NewSkipList()

	// Insert single node
	sl.Insert(10.0, "only")

	if sl.Length != 1 {
		t.Errorf("Expected length 1 after insert, got %d", sl.Length)
	}
	if sl.Tail.Score != 10.0 || sl.Tail.Member != "only" {
		t.Error("Tail should point to the only node")
	}

	// Delete the single node
	result := sl.Delete(10.0, "only")

	if !result {
		t.Error("Delete should return true for existing node")
	}
	if sl.Length != 0 {
		t.Errorf("Expected length 0 after delete, got %d", sl.Length)
	}
	if sl.Header.Forward[0] != nil {
		t.Error("List should be empty after deleting only node")
	}
	if sl.Tail != sl.Header {
		t.Error("Tail should point to header when list is empty")
	}
}

func TestSkipList_Delete_FirstNode(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(30.0, "third")

	// Delete first node
	result := sl.Delete(10.0, "first")

	if !result {
		t.Error("Delete should return true for existing node")
	}
	if sl.Length != 2 {
		t.Errorf("Expected length 2 after delete, got %d", sl.Length)
	}

	// Verify remaining nodes
	current := sl.Header.Forward[0]
	if current == nil || current.Score != 20.0 || current.Member != "second" {
		t.Error("First remaining node should be 'second' with score 20.0")
	}

	current = current.Forward[0]
	if current == nil || current.Score != 30.0 || current.Member != "third" {
		t.Error("Second remaining node should be 'third' with score 30.0")
	}

	if current.Forward[0] != nil {
		t.Error("Last node should not have a next node")
	}
}

func TestSkipList_Delete_LastNode(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(30.0, "third")

	originalTail := sl.Tail
	if originalTail.Score != 30.0 || originalTail.Member != "third" {
		t.Error("Tail should initially point to 'third'")
	}

	// Delete last node
	result := sl.Delete(30.0, "third")

	if !result {
		t.Error("Delete should return true for existing node")
	}
	if sl.Length != 2 {
		t.Errorf("Expected length 2 after delete, got %d", sl.Length)
	}

	// Verify tail pointer updated
	if sl.Tail.Score != 20.0 || sl.Tail.Member != "second" {
		t.Error("Tail should now point to 'second'")
	}

	// Verify remaining structure
	current := sl.Header.Forward[0]
	if current.Score != 10.0 || current.Member != "first" {
		t.Error("First node should be 'first'")
	}

	current = current.Forward[0]
	if current.Score != 20.0 || current.Member != "second" {
		t.Error("Second node should be 'second'")
	}

	if current.Forward[0] != nil {
		t.Error("'second' should now be the last node")
	}
}

func TestSkipList_Delete_MiddleNode(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(30.0, "third")
	sl.Insert(40.0, "fourth")

	// Delete middle node
	result := sl.Delete(20.0, "second")

	if !result {
		t.Error("Delete should return true for existing node")
	}
	if sl.Length != 3 {
		t.Errorf("Expected length 3 after delete, got %d", sl.Length)
	}

	// Verify remaining structure
	expected := []struct {
		score  float64
		member string
	}{
		{10.0, "first"},
		{30.0, "third"},
		{40.0, "fourth"},
	}

	current := sl.Header.Forward[0]
	for i, exp := range expected {
		if current == nil {
			t.Fatalf("Node %d is nil", i)
		}
		if current.Score != exp.score || current.Member != exp.member {
			t.Errorf("Node %d: expected %f/%s, got %f/%s",
				i, exp.score, exp.member, current.Score, current.Member)
		}
		current = current.Forward[0]
	}

	if current != nil {
		t.Error("Should be no more nodes after expected ones")
	}
}

func TestSkipList_Delete_SameScore_DifferentMembers(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes with same score
	sl.Insert(10.0, "apple")
	sl.Insert(10.0, "banana")
	sl.Insert(10.0, "cherry")

	// Delete middle node (lexicographically)
	result := sl.Delete(10.0, "banana")

	if !result {
		t.Error("Delete should return true for existing node")
	}
	if sl.Length != 2 {
		t.Errorf("Expected length 2 after delete, got %d", sl.Length)
	}

	// Verify remaining nodes
	expected := []string{"apple", "cherry"}
	current := sl.Header.Forward[0]

	for i, expectedMember := range expected {
		if current == nil {
			t.Fatalf("Node %d is nil", i)
		}
		if current.Score != 10.0 {
			t.Errorf("Node %d: expected score 10.0, got %f", i, current.Score)
		}
		if current.Member != expectedMember {
			t.Errorf("Node %d: expected member %s, got %s", i, expectedMember, current.Member)
		}
		current = current.Forward[0]
	}
}

func TestSkipList_Delete_AllNodes(t *testing.T) {
	sl := NewSkipList()

	// Insert multiple nodes
	nodes := []struct {
		score  float64
		member string
	}{
		{10.0, "first"},
		{20.0, "second"},
		{30.0, "third"},
		{40.0, "fourth"},
		{50.0, "fifth"},
	}

	for _, n := range nodes {
		sl.Insert(n.score, n.member)
	}

	if sl.Length != 5 {
		t.Errorf("Expected length 5 after inserts, got %d", sl.Length)
	}

	// Delete all nodes in random order
	deleteOrder := []struct {
		score  float64
		member string
	}{
		{30.0, "third"},  // middle
		{10.0, "first"},  // first
		{50.0, "fifth"},  // last
		{40.0, "fourth"}, // middle
		{20.0, "second"}, // last remaining
	}

	for i, del := range deleteOrder {
		result := sl.Delete(del.score, del.member)
		if !result {
			t.Errorf("Delete %d should return true", i)
		}

		expectedLength := 5 - i - 1
		if sl.Length != int64(expectedLength) {
			t.Errorf("Delete %d: expected length %d, got %d", i, expectedLength, sl.Length)
		}
	}

	// Verify list is empty
	if sl.Length != 0 {
		t.Errorf("Final length should be 0, got %d", sl.Length)
	}
	if sl.Header.Forward[0] != nil {
		t.Error("List should be empty")
	}
}

func TestSkipList_Delete_LevelReduction(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes to potentially create multiple levels
	for i := 0; i < 10; i++ {
		sl.Insert(float64(i), "node")
	}

	originalLevel := sl.Level
	originalLength := sl.Length

	// Delete some nodes
	for i := 0; i < 5; i++ {
		result := sl.Delete(float64(i), "node")
		if !result {
			t.Errorf("Delete of node %d should return true", i)
		}
	}

	// Verify structure consistency
	if sl.Length != originalLength-5 {
		t.Errorf("Expected length %d, got %d", originalLength-5, sl.Length)
	}

	// Verify all remaining nodes are accessible
	count := 0
	current := sl.Header.Forward[0]
	for current != nil {
		count++
		current = current.Forward[0]
	}

	if count != int(sl.Length) {
		t.Errorf("Traversal count %d doesn't match Length %d", count, sl.Length)
	}

	// Level may have been reduced
	if sl.Level > originalLevel {
		t.Errorf("Level should not increase after deletes. Was %d, now %d", originalLevel, sl.Level)
	}
}

func TestSkipList_Delete_ForwardPointersConsistency(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes
	scores := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for _, score := range scores {
		sl.Insert(score, "node")
	}

	// Delete middle node
	sl.Delete(30.0, "node")

	// Verify forward pointers are consistent across all levels
	for level := 0; level < sl.Level; level++ {
		current := sl.Header
		var prevScore float64 = -1

		for current.Forward[level] != nil {
			current = current.Forward[level]

			// Verify scores are in ascending order
			if current.Score < prevScore {
				t.Errorf("Level %d: scores not in ascending order after delete. Prev: %f, Current: %f",
					level, prevScore, current.Score)
			}

			// Verify deleted node is not present
			if current.Score == 30.0 {
				t.Errorf("Level %d: deleted node still present", level)
			}

			prevScore = current.Score
		}
	}
}

// Benchmark tests for Delete
func BenchmarkSkipList_Delete(b *testing.B) {
	sl := NewSkipList()

	// Pre-populate with nodes
	for i := 0; i < b.N; i++ {
		sl.Insert(float64(i), "node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Delete(float64(i), "node")
	}
}

func BenchmarkSkipList_Delete_Random(b *testing.B) {
	sl := NewSkipList()

	// Pre-populate with nodes
	scores := make([]float64, b.N)
	for i := 0; i < b.N; i++ {
		score := float64(i * 7 % 1000)
		scores[i] = score
		sl.Insert(score, "node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Delete(scores[i], "node")
	}
}

// ===== SEARCH TESTS =====

func TestSkipList_Search_EmptyList(t *testing.T) {
	sl := NewSkipList()

	// Search in empty list
	result := sl.Search(10.0, "test")

	if result != nil {
		t.Error("Search should return nil for empty list")
	}
}

func TestSkipList_Search_SingleNode_Found(t *testing.T) {
	sl := NewSkipList()

	// Insert single node
	inserted := sl.Insert(10.0, "test")

	// Search for the node
	result := sl.Search(10.0, "test")

	if result == nil {
		t.Fatal("Search should find the existing node")
	}
	if result != inserted {
		t.Error("Search should return the same node that was inserted")
	}
	if result.Score != 10.0 {
		t.Errorf("Expected score 10.0, got %f", result.Score)
	}
	if result.Member != "test" {
		t.Errorf("Expected member 'test', got '%s'", result.Member)
	}
}

func TestSkipList_Search_SingleNode_NotFound(t *testing.T) {
	sl := NewSkipList()

	// Insert single node
	sl.Insert(10.0, "test")

	// Search for different nodes
	testCases := []struct {
		score  float64
		member string
		desc   string
	}{
		{5.0, "test", "score too low"},
		{15.0, "test", "score too high"},
		{10.0, "other", "correct score, wrong member"},
		{5.0, "other", "both score and member wrong"},
	}

	for _, tc := range testCases {
		result := sl.Search(tc.score, tc.member)
		if result != nil {
			t.Errorf("Search should return nil for %s", tc.desc)
		}
	}
}

func TestSkipList_Search_MultipleNodes(t *testing.T) {
	sl := NewSkipList()

	// Insert multiple nodes
	testNodes := []struct {
		score  float64
		member string
	}{
		{10.0, "first"},
		{20.0, "second"},
		{30.0, "third"},
		{25.0, "middle"},
		{15.0, "between"},
	}

	insertedNodes := make(map[string]*SkipListNode)
	for _, node := range testNodes {
		inserted := sl.Insert(node.score, node.member)
		insertedNodes[node.member] = inserted
	}

	// Search for each inserted node
	for _, node := range testNodes {
		result := sl.Search(node.score, node.member)
		if result == nil {
			t.Errorf("Should find node %s with score %f", node.member, node.score)
			continue
		}

		expected := insertedNodes[node.member]
		if result != expected {
			t.Errorf("Search returned different node for %s", node.member)
		}
		if result.Score != node.score {
			t.Errorf("Node %s: expected score %f, got %f", node.member, node.score, result.Score)
		}
		if result.Member != node.member {
			t.Errorf("Node %s: expected member %s, got %s", node.member, node.member, result.Member)
		}
	}
}

func TestSkipList_Search_SameScore_DifferentMembers(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes with same score
	members := []string{"apple", "banana", "cherry", "date"}
	score := 10.0

	insertedNodes := make(map[string]*SkipListNode)
	for _, member := range members {
		inserted := sl.Insert(score, member)
		insertedNodes[member] = inserted
	}

	// Search for each member
	for _, member := range members {
		result := sl.Search(score, member)
		if result == nil {
			t.Errorf("Should find member %s", member)
			continue
		}

		if result.Score != score {
			t.Errorf("Member %s: expected score %f, got %f", member, score, result.Score)
		}
		if result.Member != member {
			t.Errorf("Member %s: expected member %s, got %s", member, member, result.Member)
		}
		if result != insertedNodes[member] {
			t.Errorf("Search returned different node for member %s", member)
		}
	}
}

func TestSkipList_Search_NonExistentNodes(t *testing.T) {
	sl := NewSkipList()

	// Insert some nodes
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(30.0, "third")

	// Search for non-existent nodes
	testCases := []struct {
		score  float64
		member string
		desc   string
	}{
		{5.0, "before", "score before all"},
		{35.0, "after", "score after all"},
		{15.0, "missing", "score between existing"},
		{20.0, "wrong", "existing score, wrong member"},
	}

	for _, tc := range testCases {
		result := sl.Search(tc.score, tc.member)
		if result != nil {
			t.Errorf("Search should return nil for %s (score: %f, member: %s)",
				tc.desc, tc.score, tc.member)
		}
	}
}

// ===== GETRANK TESTS =====

func TestSkipList_GetRank_EmptyList(t *testing.T) {
	sl := NewSkipList()

	// GetRank in empty list
	rank := sl.GetRank(10.0, "test")

	if rank != -1 {
		t.Errorf("GetRank should return -1 for empty list, got %d", rank)
	}
}

func TestSkipList_GetRank_SingleNode(t *testing.T) {
	sl := NewSkipList()

	// Insert single node
	sl.Insert(10.0, "only")

	// Get rank of existing node
	rank := sl.GetRank(10.0, "only")
	if rank != 0 {
		t.Errorf("Expected rank 0 for only node, got %d", rank)
	}

	// Get rank of non-existent node
	rank = sl.GetRank(20.0, "missing")
	if rank != -1 {
		t.Errorf("Expected rank -1 for non-existent node, got %d", rank)
	}
}

func TestSkipList_GetRank_MultipleNodes(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes in random order
	nodes := []struct {
		score  float64
		member string
	}{
		{30.0, "third"},
		{10.0, "first"},
		{50.0, "fifth"},
		{20.0, "second"},
		{40.0, "fourth"},
	}

	for _, node := range nodes {
		sl.Insert(node.score, node.member)
	}

	// Test ranks (should be 0-based, ordered by score)
	expectedRanks := []struct {
		score        float64
		member       string
		expectedRank int64
	}{
		{10.0, "first", 0},  // lowest score
		{20.0, "second", 1}, // second lowest
		{30.0, "third", 2},  // middle
		{40.0, "fourth", 3}, // second highest
		{50.0, "fifth", 4},  // highest score
	}

	for _, test := range expectedRanks {
		rank := sl.GetRank(test.score, test.member)
		if rank != test.expectedRank {
			t.Errorf("Member %s: expected rank %d, got %d",
				test.member, test.expectedRank, rank)
		}
	}
}

func TestSkipList_GetRank_SameScore_LexicographicalOrder(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes with same score (should be ordered lexicographically)
	score := 10.0
	members := []string{"zebra", "apple", "banana", "cherry"}

	for _, member := range members {
		sl.Insert(score, member)
	}

	// Expected order: apple(0), banana(1), cherry(2), zebra(3)
	expectedRanks := map[string]int64{
		"apple":  0,
		"banana": 1,
		"cherry": 2,
		"zebra":  3,
	}

	for member, expectedRank := range expectedRanks {
		rank := sl.GetRank(score, member)
		if rank != expectedRank {
			t.Errorf("Member %s: expected rank %d, got %d",
				member, expectedRank, rank)
		}
	}
}

func TestSkipList_GetRank_MixedScoresAndMembers(t *testing.T) {
	sl := NewSkipList()

	// Insert mix of scores and members
	nodes := []struct {
		score  float64
		member string
	}{
		{20.0, "b"}, // rank 3
		{10.0, "z"}, // rank 2
		{10.0, "a"}, // rank 1 (same score as z, but lexicographically first)
		{20.0, "a"}, // rank 4 (same score as b, but lexicographically first)
		{30.0, "x"}, // rank 5
	}

	for _, node := range nodes {
		sl.Insert(node.score, node.member)
	}

	// Expected order: (10.0,"a"):1, (10.0,"z"):2, (20.0,"a"):3, (20.0,"b"):4, (30.0,"x"):5
	expectedRanks := []struct {
		score        float64
		member       string
		expectedRank int64
	}{
		{10.0, "a", 0},
		{10.0, "z", 1},
		{20.0, "a", 2},
		{20.0, "b", 3},
		{30.0, "x", 4},
	}

	for _, test := range expectedRanks {
		rank := sl.GetRank(test.score, test.member)
		if rank != test.expectedRank {
			t.Errorf("Node (%f,%s): expected rank %d, got %d",
				test.score, test.member, test.expectedRank, rank)
		}
	}
}

func TestSkipList_GetRank_NonExistentNodes(t *testing.T) {
	sl := NewSkipList()

	// Insert some nodes
	sl.Insert(10.0, "first")
	sl.Insert(20.0, "second")
	sl.Insert(30.0, "third")

	// Test non-existent nodes
	testCases := []struct {
		score  float64
		member string
		desc   string
	}{
		{5.0, "before", "score before all"},
		{35.0, "after", "score after all"},
		{15.0, "missing", "score between existing"},
		{20.0, "wrong", "existing score, wrong member"},
		{10.0, "wrong", "existing score, wrong member"},
	}

	for _, tc := range testCases {
		rank := sl.GetRank(tc.score, tc.member)
		if rank != -1 {
			t.Errorf("GetRank should return -1 for %s (score: %f, member: %s), got %d",
				tc.desc, tc.score, tc.member, rank)
		}
	}
}

func TestSkipList_GetRank_AfterInsertAndDelete(t *testing.T) {
	sl := NewSkipList()

	// Insert nodes
	sl.Insert(10.0, "a") // rank 0
	sl.Insert(20.0, "b") // rank 1
	sl.Insert(30.0, "c") // rank 2
	sl.Insert(40.0, "d") // rank 3

	// Verify initial ranks
	if rank := sl.GetRank(10.0, "a"); rank != 0 {
		t.Errorf("Expected rank 0 for 'a', got %d", rank)
	}
	if rank := sl.GetRank(30.0, "c"); rank != 2 {
		t.Errorf("Expected rank 2 for 'c', got %d", rank)
	}

	// Delete middle node
	sl.Delete(20.0, "b")

	// Ranks should shift: a(0), c(1), d(2)
	expectedAfterDelete := []struct {
		score        float64
		member       string
		expectedRank int64
	}{
		{10.0, "a", 0},
		{30.0, "c", 1}, // shifted down
		{40.0, "d", 2}, // shifted down
	}

	for _, test := range expectedAfterDelete {
		rank := sl.GetRank(test.score, test.member)
		if rank != test.expectedRank {
			t.Errorf("After delete, node (%f,%s): expected rank %d, got %d",
				test.score, test.member, test.expectedRank, rank)
		}
	}

	// Deleted node should not be found
	if rank := sl.GetRank(20.0, "b"); rank != -1 {
		t.Errorf("Deleted node should return rank -1, got %d", rank)
	}
}

// Benchmark tests for Search and GetRank
func BenchmarkSkipList_Search(b *testing.B) {
	sl := NewSkipList()

	// Pre-populate with nodes
	for i := 0; i < 1000; i++ {
		sl.Insert(float64(i), "node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Search(float64(i%1000), "node")
	}
}

func BenchmarkSkipList_GetRank(b *testing.B) {
	sl := NewSkipList()

	// Pre-populate with nodes
	for i := 0; i < 1000; i++ {
		sl.Insert(float64(i), "node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.GetRank(float64(i%1000), "node")
	}
}

func BenchmarkSkipList_Search_Random(b *testing.B) {
	sl := NewSkipList()

	// Pre-populate with nodes
	scores := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		score := float64(i * 7 % 1000)
		scores[i] = score
		sl.Insert(score, "node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Search(scores[i%1000], "node")
	}
}
