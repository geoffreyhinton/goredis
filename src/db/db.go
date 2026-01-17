package db

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/geoffreyhinton/goredis/src/datastruct/dict/dict"
	List "github.com/geoffreyhinton/goredis/src/datastruct/list"
	"github.com/geoffreyhinton/goredis/src/datastruct/lock"
	"github.com/geoffreyhinton/goredis/src/interface/redis"
	"github.com/geoffreyhinton/goredis/src/lib/logger"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

const (
	StringCode = iota // Data is []byte
	ListCode          // *list.LinkedList
	SetCode
	DictCode // *dict.Dict
	SortedSetCode
)

type DataEntity struct {
	Code uint8
	Data interface{}
	sync.RWMutex
}

type DataEntityWithKey struct {
	DataEntity
	Key string
}

// args don't include cmd line
type CmdFunc func(db *DB, args [][]byte) redis.Reply

type DB struct {
	Data *dict.Dict // key -> DataEntity
	// key -> expireTime (time.Time)
	TTLMap *dict.Dict
	// dict will ensure thread safety of its method
	// use this mutex for complicated command only, eg. rpush, incr ...
	Locks *lock.LockMap
	// TimerTask interval
	interval time.Duration
}

var cmdMap = MakeCmdMap()

func MakeCmdMap() map[string]CmdFunc {
	cmdMap := make(map[string]CmdFunc)
	/*
			PING
		Công dụng: Kiểm tra kết nối với Redis server
		Ví dụ: PING
		# Trả về: PONG
	*/
	cmdMap["ping"] = Ping
	/*
			SET
		Công dụng: Thiết lập giá trị cho một key
		Ví dụ:
		SET name "John"
		# Trả về: OK
	*/
	cmdMap["set"] = Set
	/*
			SETNX
		Công dụng: Thiết lập giá trị cho một key nếu key chưa tồn tại
		Ví dụ:
		SETNX counter 1
		# Trả về: 1 (nếu key chưa tồn tại), 0 (nếu key đã tồn tại)
	*/
	cmdMap["setnx"] = SetNX
	/*
				SETEX
		Công dụng: Set key với thời gian hết hạn (tính bằng giây)
		Ví dụ:
		SETEX session_token 3600 "abc123"
		# Set token hết hạn sau 3600 giây (1 giờ)
	*/
	cmdMap["setex"] = SetEX
	/*
				PSETEX
		Công dụng: Set key với thời gian hết hạn (tính bằng mili giây)
		Ví dụ:
		PSETEX session_token 1500 "abc123"
		# Set token hết hạn sau 1500 mili giây (1.5 giây)
	*/
	cmdMap["psetex"] = PSetEX
	/*
			MSET
		Công dụng: Thiết lập nhiều key-value cùng lúc
		Ví dụ:
		MSET name "John" age "30" city "New York"
		# Trả về: OK
	*/
	cmdMap["mset"] = MSet
	/*
			MGET
		Công dụng: Lấy giá trị của nhiều key cùng lúc
		Ví dụ:
		MGET name age city
		# Trả về: "John", "30", "New York"
	*/
	cmdMap["mget"] = MGet
	/*
			MSETNX
		Công dụng: Thiết lập nhiều key-value cùng lúc nếu tất cả các key chưa tồn tại
		Ví dụ:
		MSETNX name "John" age "30"
		# Trả về: 1 (nếu tất cả các key chưa tồn tại), 0 (nếu ít nhất một key đã tồn tại)
	*/
	cmdMap["msetnx"] = MSetNX
	cmdMap["get"] = Get
	cmdMap["del"] = Del
	cmdMap["getset"] = GetSet
	cmdMap["incr"] = Incr
	cmdMap["incrby"] = IncrBy
	cmdMap["incrbyfloat"] = IncrByFloat
	cmdMap["decr"] = Decr
	cmdMap["decrby"] = DecrBy
	cmdMap["expire"] = Expire
	cmdMap["expireat"] = ExpireAt
	cmdMap["pexpire"] = PExpire
	cmdMap["pexpireat"] = PExpireAt
	cmdMap["ttl"] = TTL
	cmdMap["pttl"] = PTTL
	cmdMap["persist"] = Persist

	cmdMap["lpush"] = LPush
	cmdMap["lpushx"] = LPushX
	cmdMap["rpush"] = RPush
	cmdMap["rpushx"] = RPushX
	cmdMap["lpop"] = LPop
	cmdMap["rpop"] = RPop
	cmdMap["rpoplpush"] = RPopLPush
	cmdMap["lrem"] = LRem
	cmdMap["llen"] = LLen
	cmdMap["lindex"] = LIndex
	cmdMap["lset"] = LSet
	cmdMap["lrange"] = LRange

	cmdMap["hset"] = HSet
	cmdMap["hsetnx"] = HSetNX
	cmdMap["hget"] = HGet
	cmdMap["hexists"] = HExists
	cmdMap["hdel"] = HDel
	cmdMap["hlen"] = HLen
	cmdMap["hmget"] = HMGet
	cmdMap["hmset"] = HMSet
	cmdMap["hkeys"] = HKeys
	cmdMap["hvals"] = HVals
	cmdMap["hgetall"] = HGetAll
	cmdMap["hincrby"] = HIncrBy
	cmdMap["hincrbyfloat"] = HIncrByFloat

	return cmdMap
}

func MakeDB() *DB {
	db := &DB{
		Data:     dict.Make(1024),
		Locks:    &lock.LockMap{},
		TTLMap:   dict.Make(512),
		interval: 5 * time.Second,
	}
	db.TimerTask()
	return db
}

func (db *DB) Exec(args [][]byte) (result redis.Reply) {
	defer func() {
		if err := recover(); err != nil {
			logger.Warn(fmt.Sprintf("error occurs: %v\n%s", err, string(debug.Stack())))
			result = &reply.UnknownErrReply{}
		}
	}()

	cmd := strings.ToLower(string(args[0]))
	cmdFunc, ok := cmdMap[cmd]
	if !ok {
		return reply.MakeErrReply("ERR unknown command '" + cmd + "'")
	}
	if len(args) > 1 {
		result = cmdFunc(db, args[1:])
	} else {
		result = cmdFunc(db, [][]byte{})
	}
	return
}

func (db *DB) Get(key string) (*DataEntity, bool) {
	raw, ok := db.Data.Get(key)
	if !ok {
		return nil, false
	}
	if db.IsExpired(key) {
		return nil, false
	}
	entity, _ := raw.(*DataEntity)
	return entity, true
}
func (db *DB) Expire(key string, expireTime time.Time) {
	db.TTLMap.Put(key, expireTime)
}
func (db *DB) Persist(key string) {
	db.TTLMap.Remove(key)
}

func (db *DB) IsExpired(key string) bool {
	rawExpireTime, ok := db.TTLMap.Get(key)
	if !ok {
		return false
	}
	expireTime, _ := rawExpireTime.(time.Time)
	expired := time.Now().After(expireTime)
	if expired {
		db.Data.Remove(key)
	}
	return expired
}
func (db *DB) Remove(key string) {
	db.Data.Remove(key)
	db.TTLMap.Remove(key)
	db.Locks.Clean(key)
}

func (db *DB) Removes(keys ...string) (deleted int) {
	db.Locks.Locks(keys...)
	defer func() {
		db.Locks.UnLocks(keys...)
		db.Locks.Cleans(keys...)
	}()
	deleted = 0
	for _, key := range keys {
		_, exists := db.Data.Get(key)
		if exists {
			db.Data.Remove(key)
			db.TTLMap.Remove(key)
			deleted++
		}
	}
	return deleted
}

func (db *DB) cleanExpired() {
	now := time.Now()
	toRemove := &List.LinkedList{}
	db.TTLMap.ForEach(func(key string, val interface{}) bool {
		expireTime, _ := val.(time.Time)
		if now.After(expireTime) {
			db.Data.Remove(key)
			db.Locks.Clean(key)
			toRemove.Add(key)
		}
		return true
	})
	toRemove.ForEach(func(i int, val interface{}) bool {
		key, _ := val.(string)
		db.TTLMap.Remove(key)
		return true
	})
}

// start a goroutine to clean expired keys periodically
func (db *DB) TimerTask() {
	ticker := time.NewTicker(db.interval)
	go func() {
		for range ticker.C {
			db.cleanExpired()
		}
	}()
}
