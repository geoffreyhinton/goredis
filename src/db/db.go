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
	/*
			GET
		Công dụng: Lấy giá trị của một key
		Ví dụ:
		GET name
		# Trả về: "John"
	*/
	cmdMap["get"] = Get
	/*
			DEL
		Công dụng: Xoá một hoặc nhiều key
		Ví dụ:
		DEL name age
		# Trả về: Số lượng key đã bị xoá
	*/
	cmdMap["del"] = Del
	/*
			GETSET
		Công dụng: Thiết lập giá trị mới cho một key và trả về giá trị cũ
		Ví dụ:
		GETSET name "Doe"
		# Trả về: "John" (giá trị cũ của key "name")
	*/
	cmdMap["getset"] = GetSet
	/*
			INCR
		Công dụng: Tăng giá trị số nguyên của một key lên 1
		Ví dụ:
		INCR counter
		# Trả về: Giá trị mới của key "counter" sau khi tăng
	*/
	cmdMap["incr"] = Incr
	/*
			INCRBY
		Công dụng: Tăng giá trị số nguyên của một key lên một số nguyên được chỉ định
		Ví dụ:
		INCRBY counter 5
		# Trả về: Giá trị mới của key "counter" sau khi tăng
	*/
	cmdMap["incrby"] = IncrBy
	/*
			INCRBYFLOAT
		Công dụng: Tăng giá trị số thực của một key lên một số thực được chỉ định
		Ví dụ:
		INCRBYFLOAT balance 10.5
		# Trả về: Giá trị mới của key "balance" sau khi tăng
	*/
	cmdMap["incrbyfloat"] = IncrByFloat
	/*
			DECR
		Công dụng: Giảm giá trị số nguyên của một key đi 1
		Ví dụ:
		DECR counter
		# Trả về: Giá trị mới của key "counter" sau khi giảm
	*/
	cmdMap["decr"] = Decr
	/*
			DECRBY
		Công dụng: Giảm giá trị số nguyên của một key đi một số nguyên được chỉ định
		Ví dụ:
		DECRBY counter 3
		# Trả về: Giá trị mới của key "counter" sau khi giảm
	*/
	cmdMap["decrby"] = DecrBy
	/*
			EXPIRE
		Công dụng: Đặt thời gian hết hạn cho một key (tính bằng giây)
		Ví dụ:
		EXPIRE session_token 3600
		# Đặt thời gian hết hạn cho key "session_token" sau 3600 giây (1 giờ)
	*/
	cmdMap["expire"] = Expire
	/*
			EXPIREAT
		Công dụng: Đặt thời gian hết hạn cho một key dựa trên thời gian Unix timestamp
		Ví dụ:
		EXPIREAT session_token 1672531199
		# Đặt thời gian hết hạn cho key "session_token" vào thời điểm Unix timestamp 1672531199
	*/
	cmdMap["expireat"] = ExpireAt
	/*
			PEXPIRE
		Công dụng: Đặt thời gian hết hạn cho một key (tính bằng mili giây)
		Ví dụ:
		PEXPIRE session_token 1500
		# Đặt thời gian hết hạn cho key "session_token" sau 1500 mili giây (1.5 giây)
	*/
	cmdMap["pexpire"] = PExpire
	/*
			PEXPIREAT
		Công dụng: Đặt thời gian hết hạn cho một key dựa trên thời gian Unix timestamp (tính bằng mili giây)
		Ví dụ:
		PEXPIREAT session_token 1672531199123
		# Đặt thời gian hết hạn cho key "session_token" vào thời điểm Unix timestamp 1672531199123
	*/
	cmdMap["pexpireat"] = PExpireAt
	/*
			TTL
		Công dụng: Lấy thời gian còn lại trước khi một key hết hạn (tính bằng giây)
		Ví dụ:
		TTL session_token
		# Trả về: Số giây còn lại trước khi key "session_token" hết hạn
	*/
	cmdMap["ttl"] = TTL
	/*
			PTTL
		Công dụng: Lấy thời gian còn lại trước khi một key hết hạn (tính bằng mili giây)
		Ví dụ:
		PTTL session_token
		# Trả về: Số mili giây còn lại trước khi key "session_token" hết hạn
	*/
	cmdMap["pttl"] = PTTL
	/*
			PERSIST
		Công dụng: Loại bỏ thời gian hết hạn của một key, biến nó thành key vĩnh viễn
		Ví dụ:
		PERSIST session_token
		# Trả về: 1 (nếu thời gian hết hạn đã được loại bỏ), 0 (nếu key không tồn tại hoặc không có thời gian hết hạn)
	*/
	cmdMap["persist"] = Persist
	/*
			lpush
		Công dụng: Các lệnh thao tác với cấu trúc dữ liệu danh sách (list)
		Ví dụ:
			LPUSH mylist "world"
			LPUSH mylist "hello"
		# Trả về: 2 (độ dài mới của danh sách sau khi chèn)

	*/
	cmdMap["lpush"] = LPush
	/*
			lpushx
		Công dụng: Chèn các phần tử vào đầu danh sách chỉ khi danh sách đã tồn tại
		Ví dụ:
			LPUSHX mylist "!"
		# Trả về:  (độ dài mới của danh sách sau khi chèn)
	*/
	cmdMap["lpushx"] = LPushX
	/*
			rpush
		Công dụng: Chèn các phần tử vào cuối danh sách
		Ví dụ:
			RPUSH mylist "hello"
			RPUSH mylist "world"
		# Trả về: 2 (độ dài mới của danh sách sau khi chèn)
	*/
	cmdMap["rpush"] = RPush
	/*
			rpushx
		Công dụng: Chèn các phần tử vào cuối danh sách chỉ khi danh sách đã tồn tại
		Ví dụ:
			RPUSHX mylist "!"
		# Trả về: (độ dài mới của danh sách sau khi chèn)
	*/
	cmdMap["rpushx"] = RPushX
	/*
			lpop
		Công dụng: Loại bỏ và trả về phần tử đầu tiên của danh sách
		Ví dụ:
			LPOP mylist
		# Trả về: "hello" (phần tử bị loại bỏ)
	*/
	cmdMap["lpop"] = LPop
	/*
			rpop
		Công dụng: Loại bỏ và trả về phần tử cuối cùng của danh sách
		Ví dụ:
			RPOP mylist
		# Trả về: "world" (phần tử bị loại bỏ)
	*/
	cmdMap["rpop"] = RPop
	/*
			rpoplpush
		Công dụng: Loại bỏ phần tử cuối cùng của danh sách nguồn và chèn nó vào đầu danh sách đích
		Ví dụ:
			RPOPLPUSH sourceList destList
		# Trả về: Phần tử bị di chuyển từ sourceList sang destList
	*/
	cmdMap["rpoplpush"] = RPopLPush
	/*
			lrem
		Công dụng: Loại bỏ các phần tử khỏi danh sách dựa trên giá trị
		Ví dụ:
			LREM mylist 2 "hello"
	*/
	cmdMap["lrem"] = LRem
	/*
			llen
		Công dụng: Lấy độ dài của danh sách
		Ví dụ:
			LLEN mylist
		# Trả về: Độ dài của danh sách mylist
	*/
	cmdMap["llen"] = LLen
	/*
			lindex
		Công dụng: Lấy phần tử tại một vị trí cụ thể trong danh sách
		Ví dụ:
			LINDEX mylist 0
		# Trả về: Phần tử tại vị trí 0 trong danh sách mylist
	*/
	cmdMap["lindex"] = LIndex
	/*
			lset
		Công dụng: Cập nhật phần tử tại một vị trí cụ thể trong danh sách
		Ví dụ:
			LSET mylist 0 "newValue"
	*/
	cmdMap["lset"] = LSet
	/*
			lrange
		Công dụng: Lấy một dải phần tử từ danh sách
		Ví dụ:
			LRANGE mylist 0 -1
		# Trả về: Tất cả các phần tử trong danh sách mylist
	*/
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
