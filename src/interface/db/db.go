package db

import "github.com/geoffreyhinton/goredis/src/interface/redis"

type DB interface {
	Exec([][]byte) redis.Reply
}
