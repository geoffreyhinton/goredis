# SortedSet Implementation - Redis Sorted Sets Clone

## 1. Overview

SortedSet là implementation của Redis Sorted Sets trong GoRedis, kết hợp **Skip List** và **Hash Table** để cung cấp:
- **O(1) score lookup** qua hash table
- **O(log n) ordered operations** qua skip list
- **Range queries** hiệu quả
- **Automatic sorting** theo score

## 2. Data Structure Design

### Core Structure:
```go
type SortedSet struct {
    Dict *dict.Dict             // member -> score mapping (O(1) lookup)
    ZSL  *skiplist.SkipList     // ordered skip list (O(log n) range ops)
}
```

### Dual Storage Benefits:
| Operation | Hash Table | Skip List | Combined Result |
|-----------|------------|-----------|-----------------|
| ZSCORE member | ✅ O(1) | ❌ O(log n) | **O(1)** |
| ZADD score member | ✅ O(1) | ✅ O(log n) | **O(log n)** |
| ZRANGE start stop | ❌ N/A | ✅ O(log n + k) | **O(log n + k)** |
| ZRANK member | ❌ N/A | ✅ O(log n) | **O(log n)** |

## 3. ZADD Implementation Details

### ZADD Options:
```go
type ZAddOptions struct {
    NX   bool // Only add new elements
    XX   bool // Only update existing elements  
    CH   bool // Return number of changed elements
    INCR bool // Increment score instead of set
    GT   bool // Only update if new score > current score
    LT   bool // Only update if new score < current score
}
```

### ZADD Algorithm:
```go
func (zs *SortedSet) Add(score float64, member string, options *ZAddOptions) *ZAddResult {
    // 1. Check if member exists in hash table
    oldScoreRaw, exists := zs.Dict.Get(member)
    
    if exists {
        oldScore := oldScoreRaw.(float64)
        
        // 2. Apply option constraints (NX, XX, GT, LT)
        if shouldSkipUpdate(options, oldScore, score) {
            return &ZAddResult{Added: false, Updated: false}
        }
        
        // 3. Remove old entry from skip list
        zs.ZSL.Delete(oldScore, member)
        
        // 4. Insert new entry
        newScore := calculateNewScore(oldScore, score, options.INCR)
        zs.ZSL.Insert(newScore, member)
        zs.Dict.Put(member, newScore)
        
        return &ZAddResult{Added: false, Updated: true}
    } else {
        // New member
        zs.ZSL.Insert(score, member)
        zs.Dict.Put(member, score)
        return &ZAddResult{Added: true, Updated: false}
    }
}
```

## 4. Supported Commands

### Basic Operations:
```go
// ZADD key score member [score member ...]
ZADD leaderboard 100 "Alice" 200 "Bob" 150 "Charlie"

// ZSCORE key member
ZSCORE leaderboard "Alice"  // Returns: "100"

// ZCARD key
ZCARD leaderboard  // Returns: 3

// ZREM key member [member ...]
ZREM leaderboard "Alice"  // Returns: 1
```

### Range Operations:
```go
// ZRANGE key start stop [WITHSCORES]
ZRANGE leaderboard 0 -1 WITHSCORES
// Returns: ["Alice", "100", "Charlie", "150", "Bob", "200"]

// ZRANGEBYSCORE key min max [WITHSCORES] [LIMIT offset count]
ZRANGEBYSCORE leaderboard 100 180 WITHSCORES
// Returns: ["Alice", "100", "Charlie", "150"]

// ZRANK key member (0-based rank)
ZRANK leaderboard "Charlie"  // Returns: 1

// ZREVRANK key member (reverse rank)
ZREVRANK leaderboard "Charlie"  // Returns: 1
```

### Advanced Operations:
```go
// ZINCRBY key increment member
ZINCRBY leaderboard 50 "Alice"  // Score: 100 + 50 = 150

// ZPOPMAX key [count] - Remove highest score
ZPOPMAX leaderboard  // Returns: ["Bob", "200"]

// ZPOPMIN key [count] - Remove lowest score  
ZPOPMIN leaderboard  // Returns: ["Alice", "100"]
```

## 5. Implementation Examples

### Gaming Leaderboard:
```go
// Add players
zadd := [][]byte{
    []byte("zadd"), []byte("leaderboard"),
    []byte("2500"), []byte("Alice"),
    []byte("3200"), []byte("Bob"), 
    []byte("1800"), []byte("Charlie"),
}
db.Exec(zadd)

// Get top 3 players
zrevrange := [][]byte{
    []byte("zrevrange"), []byte("leaderboard"), 
    []byte("0"), []byte("2"), []byte("WITHSCORES"),
}
result := db.Exec(zrevrange)
// Result: ["Bob", "3200", "Alice", "2500", "Charlie", "1800"]

// Update player score
zincrby := [][]byte{
    []byte("zincrby"), []byte("leaderboard"),
    []byte("500"), []byte("Charlie"),
}
db.Exec(zincrby)  // Charlie: 1800 + 500 = 2300
```

### Priority Task Queue:
```go
// Add tasks (lower score = higher priority)
zadd := [][]byte{
    []byte("zadd"), []byte("tasks"),
    []byte("1"), []byte("critical_bug"),
    []byte("5"), []byte("feature_request"),
    []byte("10"), []byte("documentation"),
}
db.Exec(zadd)

// Get highest priority task
zpopmin := [][]byte{
    []byte("zpopmin"), []byte("tasks"),
}
result := db.Exec(zpopmin)
// Result: ["critical_bug", "1"]

// Check remaining tasks
zrange := [][]byte{
    []byte("zrange"), []byte("tasks"),
    []byte("0"), []byte("-1"), []byte("WITHSCORES"),
}
result = db.Exec(zrange)
// Result: ["feature_request", "5", "documentation", "10"]
```

### Time-based Events:
```go
// Add events with timestamps
zadd := [][]byte{
    []byte("zadd"), []byte("timeline"),
    []byte("1641234567"), []byte("event:login"),
    []byte("1641234890"), []byte("event:purchase"),
    []byte("1641235000"), []byte("event:logout"),
}
db.Exec(zadd)

// Get events in time range
zrangebyscore := [][]byte{
    []byte("zrangebyscore"), []byte("timeline"),
    []byte("1641234500"), []byte("1641234900"),
}
result := db.Exec(zrangebyscore)
// Result: ["event:login", "event:purchase"]
```

## 6. Performance Characteristics

### Time Complexity:
| Command | Complexity | Notes |
|---------|------------|-------|
| ZADD | O(log N) | Insert into skip list |
| ZSCORE | O(1) | Hash table lookup |
| ZCARD | O(1) | Stored in skip list length |
| ZRANK | O(log N) | Skip list traversal |
| ZRANGE | O(log N + K) | N=total, K=returned |
| ZRANGEBYSCORE | O(log N + K) | With score filtering |
| ZREM | O(log N) | Delete from skip list |
| ZINCRBY | O(log N) | Delete + Insert |
| ZPOPMAX/MIN | O(log N) | Access tail/head + delete |

### Space Complexity:
- **Hash Table**: O(N) - one entry per member
- **Skip List**: O(N * log N) expected - multiple levels per node
- **Total**: O(N * log N) expected space

## 7. Advanced Features

### ZADD Options Usage:
```go
// Only add if not exists
ZADD scores NX 100 "new_player"

// Only update existing  
ZADD scores XX 150 "existing_player"

// Only if new score is greater
ZADD scores GT 200 "player"  

// Only if new score is less
ZADD scores LT 50 "player"

// Increment mode
ZADD scores INCR 10 "player"  // Equivalent to ZINCRBY

// Count changed elements
ZADD scores CH 100 "p1" 200 "p2"  // Returns changed count
```

### Range Query Optimization:
```go
// ZRANGE supports negative indices
ZRANGE leaderboard -3 -1  // Last 3 elements

// ZRANGEBYSCORE with LIMIT
ZRANGEBYSCORE scores 100 200 LIMIT 0 5  // First 5 in range

// ZREVRANGE for descending order
ZREVRANGE leaderboard 0 9 WITHSCORES  // Top 10 with scores
```

## 8. Error Handling

### Common Error Cases:
```go
// Wrong argument count
ZADD key          // ERR wrong number of arguments
ZADD key 100      // ERR wrong number of arguments

// Invalid score
ZADD key abc "member"  // ERR value is not a valid float

// Type mismatch
SET mykey "string"
ZADD mykey 100 "member"  // WRONGTYPE Operation against wrong type

// Conflicting options
ZADD key NX XX 100 "member"  // ERR NX and XX not compatible
```

### Implementation Error Handling:
```go
func ZAdd(db *DB, args [][]byte) redis.Reply {
    // Validate argument count
    if len(args) < 3 {
        return reply.MakeErrReply("ERR wrong number of arguments for 'zadd' command")
    }
    
    // Parse and validate options
    options, pairs, err := parseZAddArgs(args[1:])
    if err != nil {
        return reply.MakeErrReply("ERR " + err.Error())
    }
    
    // Validate conflicting options
    if options.NX && options.XX {
        return reply.MakeErrReply("ERR NX and XX options at the same time are not compatible")
    }
    
    // Type check
    if entity.Code != SortedSetCode {
        return &reply.WrongTypeErrReply{}
    }
}
```

## 9. Memory Management

### Automatic Cleanup:
```go
// Remove empty sorted sets
if zset.Card() == 0 {
    db.Remove(key)  // Delete key entirely
}

// Node cleanup in skip list
func (sl *SkipList) Delete(score float64, member string) bool {
    // Remove from all levels
    // Update level count if necessary
    // Clean up forward pointers
}
```

### Memory Efficiency Tips:
- Use appropriate `maxLevel` (16 is good for most cases)
- Consider `skipListP` value (0.25 balances space vs performance)
- Monitor memory usage for large sorted sets
- Implement TTL for time-based cleanup

## 10. Use Case Patterns

### Pattern 1: Leaderboard with Pagination
```go
// Get page 2 (10 players per page)
start := (page - 1) * pageSize  // start = 10
stop := start + pageSize - 1    // stop = 19
ZREVRANGE leaderboard 10 19 WITHSCORES
```

### Pattern 2: Recent Activity Feed  
```go
// Add with timestamp score
timestamp := time.Now().Unix()
ZADD activity {timestamp} "user:123:action"

// Get last 50 activities  
ZREVRANGE activity 0 49
```

### Pattern 3: Rate Limiting with Score
```go
// Score = allowed time
allowedTime := time.Now().Add(time.Hour).Unix()
ZADD rate_limit {allowedTime} "user:123"

// Check if user is rate limited
score := ZSCORE rate_limit "user:123"
if score > time.Now().Unix() {
    // Still rate limited
}
```

### Pattern 4: Weighted Random Selection
```go
// Add items with weights as scores
ZADD items 10 "common_item" 5 "rare_item" 1 "legendary_item"

// Random selection based on cumulative weights
totalWeight := ZCARD items  // Sum of all scores
randomValue := rand.Float64() * totalWeight
ZRANGEBYSCORE items 0 randomValue LIMIT 0 1
```

**Kết luận**: SortedSet implementation này cung cấp đầy đủ tính năng của Redis Sorted Sets với performance tối ưu nhờ việc kết hợp Skip List và Hash Table, phù hợp cho các use cases từ gaming leaderboards đến task queues và time-series data.
