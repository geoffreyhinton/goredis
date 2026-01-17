# ZAdd Command Implementation Guide

## Overview

ZAdd is one of the most important Redis commands for working with sorted sets. Here's how it would be implemented in GoRedis.

## Data Structure Requirements

### Skip List Implementation

```go
// SkipListNode represents a node in the skip list
type SkipListNode struct {
    Score  float64
    Member string
    Forward []*SkipListNode
}

// SkipList implements the sorted set data structure
type SkipList struct {
    Header *SkipListNode
    Tail   *SkipListNode
    Length int64
    Level  int
}

// SortedSet combines skip list with hash table for O(1) member lookup
type SortedSet struct {
    Dict *dict.Dict    // member -> score mapping
    ZSL  *SkipList     // ordered skip list
}
```

### ZAdd Command Structure

```go
func ZAdd(db *DB, args [][]byte) redis.Reply {
    if len(args) < 3 {
        return reply.MakeErrReply("ERR wrong number of arguments for 'zadd' command")
    }
    
    key := string(args[0])
    
    // Parse options and score-member pairs
    options := parseZAddOptions(args[1:])
    pairs := parseScoreMemberPairs(args, options)
    
    // Get or create sorted set
    entity, exists := db.Get(key)
    if !exists {
        entity = &DataEntity{
            Code: SortedSetCode,
            Data: NewSortedSet(),
        }
        db.Data.Put(key, entity)
    }
    
    // Type check
    if entity.Code != SortedSetCode {
        return &reply.WrongTypeErrReply{}
    }
    
    zset := entity.Data.(*SortedSet)
    
    added := 0
    updated := 0
    
    // Process each score-member pair
    for _, pair := range pairs {
        result := zset.Add(pair.Score, pair.Member, options)
        if result.Added {
            added++
        }
        if result.Updated {
            updated++
        }
    }
    
    if options.CH {
        return reply.MakeIntReply(int64(added + updated))
    }
    return reply.MakeIntReply(int64(added))
}
```

## Real-World Usage Examples

### 1. Gaming Leaderboard

```redis
# Add player scores
ZADD leaderboard 2500 "Alice" 3200 "Bob" 1800 "Charlie"

# Update player score
ZADD leaderboard 2700 "Alice"

# Get top 10 players
ZREVRANGE leaderboard 0 9 WITHSCORES

# Get player rank
ZREVRANK leaderboard "Alice"
```

### 2. Time-based Feed

```redis
# Add posts with timestamps
ZADD timeline 1699123456 "post:1" 1699123789 "post:2"

# Get recent posts
ZREVRANGEBYSCORE timeline +inf -inf LIMIT 0 10

# Remove old posts
ZREMRANGEBYSCORE timeline -inf (now-86400)
```

### 3. Priority Task Queue

```redis
# Add tasks with priorities (lower = higher priority)
ZADD tasks 1 "urgent_bug_fix" 3 "feature_request" 5 "documentation"

# Get highest priority task
ZPOPMIN tasks

# Update task priority
ZADD tasks 2 "feature_request"
```

### 4. Auto-completion System

```redis
# Add search terms with popularity scores
ZADD search_terms 100 "redis" 85 "database" 92 "nosql"

# Get suggestions starting with "re"
ZRANGEBYLEX search_terms "[re" "[rf"
```

## Performance Characteristics

### Time Complexity
- **ZADD**: O(log(N)) per element
- **ZRANGE**: O(log(N) + M) where M is result size
- **ZRANK**: O(log(N))
- **ZSCORE**: O(1)

### Memory Usage
- **Skip List**: ~24 bytes per node overhead
- **Hash Table**: ~16 bytes per entry
- **Total**: ~40 bytes + key/value size per member

## Advanced Features

### Conditional Updates

```redis
# Only add if doesn't exist
ZADD scores NX 100 "new_player"

# Only update existing
ZADD scores XX 150 "existing_player"

# Only if new score is greater
ZADD scores GT 200 "player"

# Only if new score is less
ZADD scores LT 50 "player"
```

### Increment Mode

```redis
# Increment player score by 50
ZADD scores INCR 50 "player"
"150"

# This is equivalent to ZINCRBY
ZINCRBY scores 50 "player"
```

### Bulk Operations

```redis
# Add multiple members at once
ZADD products 4.5 "laptop" 3.2 "mouse" 4.8 "keyboard" 2.1 "cable"

# With change count
ZADD products CH 4.6 "laptop" 4.9 "monitor"
(integer) 1  # Only monitor was added, laptop was updated
```

## Integration with Other Commands

ZAdd works together with other sorted set commands:

```redis
# Add data
ZADD myset 1 "a" 2 "b" 3 "c"

# Query data
ZRANGE myset 0 -1           # Get all members
ZRANGEBYSCORE myset 1 2     # Get by score range
ZCARD myset                 # Get count
ZSCORE myset "b"            # Get member score

# Modify data
ZREM myset "b"              # Remove member
ZINCRBY myset 5 "a"         # Increment score
ZPOPMAX myset               # Remove highest score member
```

## Error Handling

```redis
# Wrong number of arguments
ZADD myset 1
(error) ERR wrong number of arguments for 'zadd' command

# Invalid score
ZADD myset abc "member"
(error) ERR value is not a valid float

# Wrong type
SET mykey "string"
ZADD mykey 1 "member"
(error) WRONGTYPE Operation against a key holding the wrong kind of value
```

## Best Practices

1. **Use meaningful scores**: Choose scores that represent your sorting criteria
2. **Consistent scoring**: Use same scale for all members
3. **Batch operations**: Add multiple members in single ZADD for better performance
4. **Index cleanup**: Remove outdated entries to maintain performance
5. **Memory monitoring**: Large sorted sets can consume significant memory

## Implementation Tips

1. **Skip List Levels**: Typically use 16-32 levels for good performance
2. **Score Precision**: Use IEEE 754 double precision (53-bit mantissa)
3. **Lexicographical Ordering**: Same scores are ordered lexicographically
4. **Concurrent Access**: Use appropriate locking for thread safety

This implementation would make GoRedis compatible with Redis sorted set operations!