# Skip List - Cấu trúc dữ liệu xác suất (Probabilistic Data Structure)

## 1. Khái niệm cơ bản

Skip List là một cấu trúc dữ liệu **xác suất** cho phép tìm kiếm, chèn và xóa trong **O(log n)** trung bình. Nó được thiết kế như một **"linked list có nhiều tầng"** để tăng tốc độ truy cập.

## 2. Cấu trúc Multi-Level

```
Level 3: [Header] -----------------> [30] --------> NULL
Level 2: [Header] ---------> [20] -> [30] -> [50] -> NULL  
Level 1: [Header] -> [10] -> [20] -> [30] -> [50] -> [70] -> NULL
Level 0: [Header] -> [10] -> [20] -> [30] -> [50] -> [70] -> NULL
```

**Giải thích:**
- **Level 0**: Chứa tất cả nodes (như linked list thông thường)
- **Level cao hơn**: Chỉ chứa một số nodes được chọn ngẫu nhiên
- **Header**: Node đặc biệt không chứa data, chỉ để làm điểm bắt đầu

## 3. Random Level Generation - Core của Skip List

```go
const skipListP = 0.25  // Xác suất 25%

func randomLevel() int {
    level := 1
    // 25% cơ hội lên level cao hơn
    for rnd.Float64() < 0.25 && level < maxLevel {
        level++
    }
    return level
}
```

**Phân tích xác suất:**
- Level 1: 100% nodes
- Level 2: 25% nodes  
- Level 3: 6.25% nodes (25% của 25%)
- Level 4: 1.56% nodes
- ...

## 4. Thuật toán Search

**Search Algorithm:**
```go
func Search(score, member) {
    x := header
    // Từ level cao nhất xuống level 0
    for level := maxLevel-1; level >= 0; level-- {
        // Di chuyển ngang trong level này
        while (x.forward[level] != null && x.forward[level].score < target) {
            x = x.forward[level]  // "Nhảy xa"
        }
        // Xuống level thấp hơn để tìm chính xác
    }
    return x.forward[0]  // Kết quả ở level 0
}
```

**Visualization Search cho score=30:**
```
Bước 1 (Level 3): Header -----> 30 ✓ (Found!)
Bước 2 (Level 2): Header -----> 20 -> 30 ✓  
Bước 3 (Level 1): Header -> 10 -> 20 -> 30 ✓
Bước 4 (Level 0): Header -> 10 -> 20 -> 30 ✓ (Return)
```

## 5. Thuật toán Insert

**Insert Algorithm Steps:**
1. **Search phase**: Tìm vị trí insert và lưu update pointers
2. **Random level**: Generate random level cho node mới
3. **Link phase**: Cập nhật pointers ở tất cả levels

**Ví dụ Insert score=25:**
```
Before:
Level 1: Header -> [20] -> [30] -> NULL
Level 0: Header -> [20] -> [30] -> NULL

randomLevel() = 2  // Node mới sẽ có 2 levels

After:
Level 1: Header -> [20] -> [25] -> [30] -> NULL  
Level 0: Header -> [20] -> [25] -> [30] -> NULL
```

## 6. Performance Analysis

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| **Search** | O(log n) avg, O(n) worst | O(1) |
| **Insert** | O(log n) avg, O(n) worst | O(log n) |
| **Delete** | O(log n) avg, O(n) worst | O(1) |
| **Space** | O(n) | O(n log n) expected |

## 7. Tại sao Skip List tốt cho Redis?

### A. Advantages:
- **Range queries hiệu quả**: `ZRANGE`, `ZRANGEBYSCORE`
- **Concurrent friendly**: Dễ implement lock-free hơn balanced trees
- **Simple implementation**: Không cần rotation như AVL/Red-Black trees
- **Good cache locality**: Forward pointers thường gần nhau

### B. So sánh với Binary Search Tree:
```
BST worst case: O(n) - khi cây bị skewed
Skip List worst case: O(n) - nhưng xác suất rất thấp (2^-n)

BST average: O(log n)  
Skip List average: O(log n) - guaranteed by probability
```

## 8. Redis Implementation Details

Trong Redis, Skip List được combine với **Hash Table**:
```go
type SortedSet struct {
    Dict *dict.Dict      // member -> score (O(1) lookup)
    ZSL  *skiplist.SkipList  // ordered structure (O(log n) range)
}
```

**Dual structure benefits:**
- `ZSCORE member`: O(1) qua hash table
- `ZRANGE start stop`: O(log n + k) qua skip list  
- `ZADD`: O(log n) update cả 2 structures

## 9. Practical Example

```go
// Leaderboard implementation
sl := NewSkipList()

// Insert players
sl.Insert(2500.0, "Alice")   // Score: 2500
sl.Insert(3200.0, "Bob")     // Score: 3200  
sl.Insert(1800.0, "Charlie") // Score: 1800

// Structure tạo thành:
// Level 1: Header -> Charlie(1800) -> Alice(2500) -> Bob(3200)
// Level 0: Header -> Charlie(1800) -> Alice(2500) -> Bob(3200)

// Get top 2 players
top2 := sl.GetRange(-2, -1)  // Last 2 = highest scores
// Result: ["Bob(3200)", "Alice(2500)"]
```

## 10. Implementation Code Structure

### Node Structure:
```go
type SkipListNode struct {
    Score   float64           // Điểm số để sắp xếp
    Member  string            // Tên member
    Forward []*SkipListNode   // Array pointers đến levels khác nhau
}
```

### Skip List Structure:
```go
type SkipList struct {
    Header *SkipListNode     // Node đầu tiên (không chứa data)
    Tail   *SkipListNode     // Node cuối cùng (optimization)
    Length int64             // Số lượng elements
    Level  int               // Level cao nhất hiện tại
}
```

### Key Operations:
1. **NewSkipList()**: Tạo skip list mới với header node
2. **Insert(score, member)**: Thêm/update element
3. **Delete(score, member)**: Xóa element
4. **Search(score, member)**: Tìm kiếm element
5. **GetRank(score, member)**: Lấy thứ hạng của element

## 11. Use Cases trong Redis Sorted Sets

### Gaming Leaderboard:
```redis
ZADD leaderboard 2500 "Alice" 3200 "Bob" 1800 "Charlie"
ZREVRANGE leaderboard 0 9 WITHSCORES  # Top 10 players
```

### Priority Task Queue:
```redis
ZADD task_queue 1 "urgent" 5 "normal" 10 "low"
ZPOPMIN task_queue  # Get highest priority task
```

### Time-based Events:
```redis
ZADD events 1641234567 "event1" 1641234890 "event2"
ZRANGEBYSCORE events 0 1641234600  # Events before timestamp
```

**Kết luận**: Skip List là lựa chọn tuyệt vời cho Sorted Sets vì nó cân bằng được performance, simplicity và khả năng handle range operations efficiently - chính xác những gì Redis Sorted Sets cần.
