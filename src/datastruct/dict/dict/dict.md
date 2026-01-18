# Concurrent Hash Table Implementation - Technical Documentation

## Overview

The `dict.go` file implements a high-performance, thread-safe hash table (dictionary) designed for concurrent access in a Redis-like database system. This implementation features:

- **Lock-free reads during rehashing** using atomic operations
- **Incremental rehashing** to avoid blocking operations
- **Sharded architecture** for reduced lock contention
- **Dynamic resizing** with concurrent migration
- **FNV-1a hash function** for uniform distribution

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Data Structures](#data-structures)
3. [Hash Function & Distribution](#hash-function--distribution)
4. [Concurrency Control](#concurrency-control)
5. [Rehashing Mechanism](#rehashing-mechanism)
6. [Core Operations](#core-operations)
7. [Performance Characteristics](#performance-characteristics)
8. [Usage Patterns](#usage-patterns)

## Architecture Overview

```
Dict
├── table (atomic.Value)        → Current hash table (array of shards)
├── nextTable ([]*Shard)        → New hash table during rehashing
├── rehashIndex (int32)         → Progress indicator for rehashing
└── count (int32)              → Total number of entries

Shard (Hash bucket with chaining)
├── head (*Node)               → First node in linked list
└── mutex (sync.RWMutex)      → Reader-writer lock

Node (Key-value pair in linked list)
├── key (string)              → Entry key
├── val (interface{})         → Entry value
├── next (*Node)             → Next node in chain
└── hashCode (uint32)        → Cached hash value
```

## Data Structures

### 1. Dict Structure

```go
type Dict struct {
    table       atomic.Value // []*Shard - Current hash table
    nextTable   []*Shard     // Next table during rehashing
    nextTableMu sync.Mutex   // Protects nextTable creation
    count       int32        // Atomic counter for entries
    rehashIndex int32        // Rehashing progress (-1 = not rehashing)
}
```

**Key Design Decisions:**
- `atomic.Value` for lock-free table switching
- Separate `nextTable` for incremental rehashing
- Atomic counters for thread-safe statistics

### 2. Shard Structure

```go
type Shard struct {
    head  *Node           // Head of linked list chain
    mutex sync.RWMutex    // Reader-writer mutex for fine-grained locking
}
```

**Sharding Benefits:**
- Reduces lock contention across multiple threads
- Enables parallel access to different shards
- Balances memory overhead vs. concurrency

### 3. Node Structure

```go
type Node struct {
    key      string        // Entry key
    val      interface{}   // Entry value (Redis-compatible)
    next     *Node        // Linked list pointer
    hashCode uint32       // Cached hash for rehashing
}
```

**Design Rationale:**
- Linked list handles hash collisions via chaining
- Cached `hashCode` avoids recomputation during rehashing
- `interface{}` values support Redis' polymorphic data types

## Hash Function & Distribution

### FNV-1a Hash Algorithm

```go
func fnv32(key string) uint32 {
    hash := uint32(2166136261)  // FNV offset basis
    for i := 0; i < len(key); i++ {
        hash *= prime32         // FNV prime = 16777619
        hash ^= uint32(key[i])
    }
    return hash
}
```

**Algorithm Properties:**
- **Good distribution**: Minimizes clustering and collision patterns
- **Fast computation**: Single pass through string with simple operations
- **Deterministic**: Same key always produces same hash
- **Avalanche effect**: Small input changes cause large hash changes

### Shard Selection

```go
func (dict *Dict) spread(hashCode uint32) uint32 {
    table, _ := dict.table.Load().([]*Shard)
    tableSize := uint32(len(table))
    return (tableSize - 1) & uint32(hashCode)  // Fast modulo for powers of 2
}
```

**Optimization Techniques:**
- **Power-of-2 sizing**: Enables fast modulo using bitwise AND
- **Uniform distribution**: FNV hash ensures even shard distribution
- **Cache-friendly**: Minimizes memory access patterns

## Concurrency Control

### Multi-Level Locking Strategy

1. **Dictionary Level**: `nextTableMu` protects table creation
2. **Shard Level**: `RWMutex` enables concurrent reads
3. **Atomic Operations**: Lock-free counters and rehash coordination

### Reader-Writer Lock Usage

```go
// Read operations (Get, ForEach)
shard.mutex.RLock()
defer shard.mutex.RUnlock()

// Write operations (Put, Remove)
shard.mutex.Lock()
defer shard.mutex.Unlock()
```

**Concurrency Benefits:**
- Multiple readers can access same shard simultaneously
- Writers have exclusive access for consistency
- Independent shards can be accessed in parallel

### Lock-Free Rehashing Coordination

```go
rehashIndex := atomic.LoadInt32(&dict.rehashIndex)
if rehashIndex >= int32(index) {
    // Use next table
} else {
    // Use current table
}
```

**Atomic Coordination:**
- `rehashIndex` tracks migration progress atomically
- Threads can determine table selection without locks
- Prevents race conditions during table transitions

## Rehashing Mechanism

### Trigger Conditions

```go
func (dict *Dict) addCount() int32 {
    count := atomic.AddInt32(&dict.count, 1)
    table, _ := dict.table.Load().([]*Shard)
    if float64(count) >= float64(len(table))*loadFactor {  // loadFactor = 0.75
        dict.resize()
    }
    return count
}
```

### Incremental Migration Process

```go
func (dict *Dict) transfer(wg *sync.WaitGroup) {
    for {
        i := uint32(atomic.AddInt32(&dict.rehashIndex, 1)) - 1
        if i >= tableSize {
            wg.Done()
            return
        }
        
        // Split shard into two new shards
        shard := dict.getShard(i)
        nextShard0 := dict.nextTable[i]
        nextShard1 := dict.nextTable[i+tableSize]
        
        // Redistribute nodes based on hash bit
        for node := shard.head; node != nil; node = node.next {
            if node.hashCode&tableSize == 0 {
                // Place in first new shard
            } else {
                // Place in second new shard
            }
        }
    }
}
```

**Migration Algorithm:**
1. **Concurrent Workers**: Multiple goroutines migrate shards in parallel
2. **Atomic Progress**: `rehashIndex` coordinates work distribution
3. **Bit-Based Splitting**: Uses hash bit patterns to split nodes
4. **Lock Ordering**: Prevents deadlocks during migration

### Double-Table Lookup During Rehashing

```go
func (dict *Dict) Get(key string) (val interface{}, exists bool) {
    hashCode := fnv32(key)
    index := dict.spread(hashCode)
    rehashIndex := atomic.LoadInt32(&dict.rehashIndex)
    
    if rehashIndex >= int32(index) {
        // Shard has been migrated, check next table
        nextShard := dict.getNextShard(hashCode)
        val, exists = nextShard.Get(key)
    } else {
        // Shard not yet migrated, check current table
        shard := dict.getShard(index)
        val, exists = shard.Get(key)
    }
    return
}
```

## Core Operations

### Get Operation

**Complexity:** O(1) average, O(n) worst case
**Thread Safety:** Lock-free reads during rehashing

```go
Flow:
1. Compute FNV hash of key
2. Determine target shard index
3. Check rehash status atomically
4. Select appropriate table (current vs next)
5. Acquire read lock on target shard
6. Linear search through collision chain
7. Return result and release lock
```

### Put Operation

**Complexity:** O(1) average, O(n) worst case
**Side Effects:** May trigger rehashing

```go
Flow:
1. Compute hash and determine shard
2. Check rehash status for table selection
3. Acquire write lock on target shard
4. Search collision chain for existing key
5. Update existing node or append new node
6. Release lock and update global counter
7. Check load factor and trigger resize if needed
```

### Remove Operation

**Complexity:** O(1) average, O(n) worst case
**Implementation:** Linked list node removal

```go
Flow:
1. Hash key and select appropriate table/shard
2. Acquire write lock on target shard
3. Handle special cases:
   - Empty shard: Return 0
   - Remove head: Update shard.head pointer
   - Remove middle: Update prev.next pointer
4. Decrement global counter atomically
5. Release lock and return result
```

## Performance Characteristics

### Time Complexity

| Operation | Average | Worst Case | Notes |
|-----------|---------|------------|-------|
| Get       | O(1)    | O(n)      | n = max chain length |
| Put       | O(1)    | O(n)      | Includes collision handling |
| Remove    | O(1)    | O(n)      | Linear chain traversal |
| Rehash    | O(n)    | O(n)      | Distributed across operations |

### Space Complexity

- **Memory Overhead**: ~24 bytes per node (key, value, next, hashCode)
- **Shard Overhead**: 16 bytes per shard (head pointer, mutex)
- **Table Overhead**: 8 bytes per shard pointer
- **Total**: O(n) where n = number of entries

### Concurrency Performance

- **Read Scalability**: Near-linear with core count (RWMutex)
- **Write Throughput**: Limited by shard contention
- **Rehash Impact**: Minimal blocking due to incremental migration
- **Memory Barriers**: Atomic operations provide necessary ordering

## Usage Patterns

### Redis Integration

```go
// Typical Redis usage
db := dict.Make(1024)  // Initialize with expected capacity

// SET command
db.Put("user:123", userObject)

// GET command
value, exists := db.Get("user:123")

// DEL command
deleted := db.Remove("user:123")

// Iteration for KEYS command
db.ForEach(func(key string, val interface{}) bool {
    if matches(key, pattern) {
        results = append(results, key)
    }
    return true  // Continue iteration
})
```

### Capacity Planning

```go
// Choose initial capacity based on expected load
expectedEntries := 10000
loadFactor := 0.75
initialShards := int(float64(expectedEntries) / loadFactor)
dict := Make(initialShards)
```

### Memory Management

- **Automatic Growth**: Dictionary doubles size when load factor exceeds 0.75
- **No Shrinking**: Implementation does not automatically shrink
- **GC Friendly**: Uses pointers for nodes, enabling efficient garbage collection

## Configuration Constants

```go
const (
    maxCapacity      = 1 << 15    // 32,768 maximum shards
    minCapacity      = 16         // Minimum 16 shards
    rehashConcurrent = 4          // 4 concurrent migration workers
    loadFactor       = 0.75       // Resize trigger threshold
)
```

**Tuning Considerations:**
- **maxCapacity**: Limits memory usage and lock overhead
- **minCapacity**: Ensures reasonable concurrency even for small datasets
- **rehashConcurrent**: Balances migration speed vs CPU overhead
- **loadFactor**: Trade-off between memory usage and collision probability

## Best Practices

1. **Initialization**: Size dictionary appropriately for expected load
2. **Key Design**: Use strings that hash well (avoid common prefixes)
3. **Value Types**: Store pointers for large objects to reduce copying
4. **Error Handling**: Check for nil returns and handle gracefully
5. **Monitoring**: Track load factor and resize frequency for tuning

This concurrent hash table implementation provides excellent performance characteristics for Redis-style workloads while maintaining strong consistency guarantees through careful concurrency control and incremental rehashing mechanisms.