package db

import (
	"github.com/geoffreyhinton/goredis/src/datastruct/dict/dict"
	"github.com/geoffreyhinton/goredis/src/interface/redis"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

func HSet(db *DB, args [][]byte) redis.Reply {
	if len(args) < 3 || len(args)%2 == 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'hset' command")
	}
	key := string(args[0])
	field := string(args[1])
	value := args[2]

	db.Locks.Lock(key)
	defer db.Locks.UnLock(key)

	entity, exists := db.Get(key)
	if !exists {
		entity = &DataEntity{
			Code: DictCode,
			Data: make(map[string][]byte),
		}
		db.Data.Put(key, entity)
	}
	if entity.Code != DictCode {
		return &reply.WrongTypeErrReply{}
	}
	dict, _ := entity.Data.(*dict.Dict)
	result := dict.Put(field, value)
	return reply.MakeIntReply(int64(result))
}
