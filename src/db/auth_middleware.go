package db

import (
	"strings"

	"github.com/geoffreyhinton/goredis/src/interface/redis"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

// AuthContext holds authentication information for a request
type AuthContext struct {
	User          *User
	Authenticated bool
}

// AuthenticatedExec wraps the normal Exec with authentication checks
func (db *DB) AuthenticatedExec(ctx *AuthContext, args [][]byte) redis.Reply {
	if len(args) == 0 {
		return reply.MakeErrReply("ERR empty command")
	}

	cmd := strings.ToLower(string(args[0]))

	// AUTH command is always allowed
	if cmd == "auth" {
		result := Auth(db, args[1:])

		// If AUTH was successful, update the context
		if _, ok := result.(*reply.OkReply); ok {
			// Extract username from AUTH command
			var username string
			if len(args) == 2 {
				username = "default"
			} else if len(args) == 3 {
				username = string(args[1])
			}

			if user, exists := db.Auth.GetUser(username); exists {
				ctx.User = user
				ctx.Authenticated = true
			}
		}

		return result
	}

	// ACL commands require authentication (except for basic info)
	if cmd == "acl" {
		if !ctx.Authenticated && ctx.User.Username == "default" {
			return reply.MakeErrReply("NOAUTH Authentication required")
		}
	}

	// Check if user can execute this command
	if ctx.User != nil {
		if !ctx.User.CanExecuteCommand(cmd) {
			return reply.MakeErrReply("NOPERM this user has no permissions to run the '" + cmd + "' command")
		}

		// For commands that operate on keys, check key permissions
		if len(args) > 1 && isKeyCommand(cmd) {
			key := string(args[1])
			if !ctx.User.CanAccessKey(key) {
				return reply.MakeErrReply("NOPERM this user has no permissions to access key '" + key + "'")
			}
		}
	}

	// Execute the command normally
	return db.Exec(args)
}

// isKeyCommand returns true if the command operates on keys
func isKeyCommand(cmd string) bool {
	keyCommands := map[string]bool{
		"get":         true,
		"set":         true,
		"del":         true,
		"exists":      true,
		"expire":      true,
		"ttl":         true,
		"lpush":       true,
		"rpush":       true,
		"lpop":        true,
		"rpop":        true,
		"llen":        true,
		"lindex":      true,
		"lrange":      true,
		"lrem":        true,
		"lset":        true,
		"incr":        true,
		"decr":        true,
		"incrby":      true,
		"decrby":      true,
		"getset":      true,
		"setnx":       true,
		"setex":       true,
		"psetex":      true,
		"mget":        true,
		"mset":        true,
		"msetnx":      true,
		"lpushx":      true,
		"rpushx":      true,
		"rpoplpush":   true,
		"incrbyfloat": true,
	}

	return keyCommands[cmd]
}

// CreateAuthContext creates a new authentication context with default user
func (db *DB) CreateAuthContext() *AuthContext {
	return &AuthContext{
		User:          db.Auth.GetDefaultUser(),
		Authenticated: false,
	}
}
