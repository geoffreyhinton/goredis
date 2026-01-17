package db

import (
	"strconv"
	"time"

	"github.com/geoffreyhinton/goredis/src/interface/redis"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

func Del(db *DB, args [][]byte) redis.Reply {
	if len(args) == 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'del' command")
	}
	keys := make([]string, len(args))
	for i, v := range args {
		keys[i] = string(v)
	}
	deleted := db.Removes(keys...)
	return reply.MakeIntReply(int64(deleted))
}

func Expire(db *DB, args [][]byte) redis.Reply {
	if len(args) != 2 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'expire' command")
	}
	key := string(args[0])
	ttlArg, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return reply.MakeErrReply("ERR value is not an integer or out of range")
	}
	ttl := time.Duration(ttlArg) * time.Second

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(0)
	}
	expireTime := time.Now().Add(ttl)
	db.Expire(key, expireTime)
	return reply.MakeIntReply(1)
}

func ExpireAt(db *DB, args [][]byte) redis.Reply {
	if len(args) != 2 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'expireat' command")
	}
	key := string(args[0])
	timestampArg, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return reply.MakeErrReply("ERR value is not an integer or out of range")
	}
	expireTime := time.Unix(timestampArg, 0)

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(0)
	}
	db.Expire(key, expireTime)
	return reply.MakeIntReply(1)
}

func PExpire(db *DB, args [][]byte) redis.Reply {
	if len(args) != 2 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'pexpire' command")
	}
	key := string(args[0])
	ttlArg, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return reply.MakeErrReply("ERR value is not an integer or out of range")
	}
	ttl := time.Duration(ttlArg) * time.Millisecond

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(0)
	}
	expireTime := time.Now().Add(ttl)
	db.Expire(key, expireTime)
	return reply.MakeIntReply(1)
}

func PExpireAt(db *DB, args [][]byte) redis.Reply {
	if len(args) != 2 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'pexpireat' command")
	}
	key := string(args[0])
	timestampArg, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return reply.MakeErrReply("ERR value is not an integer or out of range")
	}
	expireTime := time.Unix(0, timestampArg*int64(time.Millisecond))

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(0)
	}
	db.Expire(key, expireTime)
	return reply.MakeIntReply(1)
}

func TTL(db *DB, args [][]byte) redis.Reply {
	if len(args) != 1 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'ttl' command")
	}
	key := string(args[0])

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(-2)
	}
	rawExpireTime, ok := db.TTLMap.Get(key)
	if !ok {
		return reply.MakeIntReply(-1)
	}
	expireTime, _ := rawExpireTime.(time.Time)
	ttl := int64(expireTime.Sub(time.Now()) / time.Second)
	if ttl < 0 {
		return reply.MakeIntReply(-2)
	}
	return reply.MakeIntReply(ttl)
}

func PTTL(db *DB, args [][]byte) redis.Reply {
	if len(args) != 1 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'pttl' command")
	}
	key := string(args[0])

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(-2)
	}
	rawExpireTime, exists := db.TTLMap.Get(key)
	if !exists {
		return reply.MakeIntReply(-1)
	}
	expireTime, _ := rawExpireTime.(time.Time)
	ttl := int64(expireTime.Sub(time.Now()) / time.Millisecond)
	return reply.MakeIntReply(ttl)
}

func Persit(db *DB, args [][]byte) redis.Reply {
	if len(args) != 1 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'persist' command")
	}
	key := string(args[0])

	_, exists := db.Get(key)
	if !exists {
		return reply.MakeIntReply(0)
	}
	_, exists = db.TTLMap.Get(key)
	if !exists {
		return reply.MakeIntReply(0)
	}
	db.TTLMap.Remove(key)
	return reply.MakeIntReply(1)
}
