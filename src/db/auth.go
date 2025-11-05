package db

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/geoffreyhinton/goredis/src/interface/redis"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

// User represents a Redis user with authentication and authorization info
type User struct {
	Username     string          `json:"username"`
	PasswordHash string          `json:"password_hash"`
	Enabled      bool            `json:"enabled"`
	NoPass       bool            `json:"nopass"`   // User can authenticate without password
	Commands     map[string]bool `json:"commands"` // Allowed commands (nil = all allowed)
	Keys         []string        `json:"keys"`     // Key patterns user can access
	CreatedAt    time.Time       `json:"created_at"`
	LastLogin    *time.Time      `json:"last_login"`
	Flags        map[string]bool `json:"flags"` // Additional user flags
}

// AuthManager manages user authentication and authorization
type AuthManager struct {
	users       map[string]*User // username -> User
	defaultUser *User            // Default user for non-authenticated connections
	mutex       sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	auth := &AuthManager{
		users: make(map[string]*User),
	}

	// Create default user (unauthenticated access)
	auth.defaultUser = &User{
		Username:  "default",
		Enabled:   true,
		NoPass:    true,
		Commands:  nil,           // All commands allowed by default
		Keys:      []string{"*"}, // All keys allowed
		CreatedAt: time.Now(),
		Flags:     make(map[string]bool),
	}

	// Create admin user with password "admin123"
	adminUser := &User{
		Username:     "admin",
		PasswordHash: hashPassword("admin123"),
		Enabled:      true,
		NoPass:       false,
		Commands:     nil,           // All commands allowed
		Keys:         []string{"*"}, // All keys allowed
		CreatedAt:    time.Now(),
		Flags:        make(map[string]bool),
	}
	auth.users["admin"] = adminUser

	return auth
}

// hashPassword creates a SHA-256 hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// VerifyPassword checks if the provided password matches the hash
func verifyPassword(password, hash string) bool {
	return hashPassword(password) == hash
}

// GetUser retrieves a user by username
func (am *AuthManager) GetUser(username string) (*User, bool) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	user, exists := am.users[username]
	return user, exists
}

// GetDefaultUser returns the default user for unauthenticated connections
func (am *AuthManager) GetDefaultUser() *User {
	return am.defaultUser
}

// CreateUser creates a new user
func (am *AuthManager) CreateUser(username, password string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if _, exists := am.users[username]; exists {
		return errors.New("User '" + username + "' already exists")
	}

	user := &User{
		Username:     username,
		PasswordHash: hashPassword(password),
		Enabled:      true,
		NoPass:       false,
		Commands:     make(map[string]bool),
		Keys:         []string{"*"},
		CreatedAt:    time.Now(),
		Flags:        make(map[string]bool),
	}

	am.users[username] = user
	return nil
}

// DeleteUser removes a user
func (am *AuthManager) DeleteUser(username string) bool {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if _, exists := am.users[username]; !exists {
		return false
	}

	delete(am.users, username)
	return true
}

// ListUsers returns all usernames
func (am *AuthManager) ListUsers() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	usernames := make([]string, 0, len(am.users))
	for username := range am.users {
		usernames = append(usernames, username)
	}
	return usernames
}

// Authenticate verifies user credentials
func (am *AuthManager) Authenticate(username, password string) (*User, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	user, exists := am.users[username]
	if !exists {
		return nil, errors.New("WRONGPASS invalid username-password pair")
	}

	if !user.Enabled {
		return nil, errors.New("User is disabled")
	}

	if user.NoPass {
		// Update last login
		now := time.Now()
		user.LastLogin = &now
		return user, nil
	}

	if !verifyPassword(password, user.PasswordHash) {
		return nil, errors.New("WRONGPASS invalid username-password pair")
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now

	return user, nil
}

// CanExecuteCommand checks if user has permission to execute a command
func (user *User) CanExecuteCommand(command string) bool {
	if user.Commands == nil {
		return true // All commands allowed
	}

	command = strings.ToLower(command)
	allowed, exists := user.Commands[command]
	if !exists {
		return false // Command not in allowed list
	}

	return allowed
}

// CanAccessKey checks if user has permission to access a key
func (user *User) CanAccessKey(key string) bool {
	if len(user.Keys) == 0 {
		return false
	}

	for _, pattern := range user.Keys {
		if pattern == "*" {
			return true
		}
		// Simple pattern matching (can be enhanced with glob patterns)
		if strings.Contains(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(key, prefix) {
				return true
			}
		} else if pattern == key {
			return true
		}
	}

	return false
}

// AUTH command implementation
func Auth(db *DB, args [][]byte) redis.Reply {
	if len(args) < 1 || len(args) > 2 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'auth' command")
	}

	var username, password string

	if len(args) == 1 {
		// AUTH password (authenticate as default user)
		username = "default"
		password = string(args[0])
	} else {
		// AUTH username password
		username = string(args[0])
		password = string(args[1])
	}

	_, err := db.Auth.Authenticate(username, password)
	if err != nil {
		return reply.MakeErrReply(err.Error())
	}

	// Authentication successful
	return &reply.OkReply{}
}

// ACL USER command implementation
func AclUser(db *DB, args [][]byte) redis.Reply {
	if len(args) < 2 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'acl user' command")
	}

	username := string(args[0])

	// Parse ACL rules
	for i := 1; i < len(args); i++ {
		rule := strings.ToLower(string(args[i]))

		switch {
		case rule == "on":
			// Enable user
			user, exists := db.Auth.GetUser(username)
			if !exists {
				// Create new user
				err := db.Auth.CreateUser(username, "")
				if err != nil {
					return reply.MakeErrReply(err.Error())
				}
				user, _ = db.Auth.GetUser(username)
				user.NoPass = true // No password required by default
			}
			user.Enabled = true

		case rule == "off":
			// Disable user
			user, exists := db.Auth.GetUser(username)
			if exists {
				user.Enabled = false
			}

		case strings.HasPrefix(rule, ">"):
			// Set password
			password := rule[1:]
			user, exists := db.Auth.GetUser(username)
			if !exists {
				err := db.Auth.CreateUser(username, password)
				if err != nil {
					return reply.MakeErrReply(err.Error())
				}
			} else {
				user.PasswordHash = hashPassword(password)
				user.NoPass = false
			}

		case rule == "nopass":
			// Allow authentication without password
			user, exists := db.Auth.GetUser(username)
			if exists {
				user.NoPass = true
			}
		}
	}

	return &reply.OkReply{}
}

// ACL LIST command implementation
func AclList(db *DB, args [][]byte) redis.Reply {
	if len(args) != 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'acl list' command")
	}

	usernames := db.Auth.ListUsers()
	result := make([][]byte, len(usernames))

	for i, username := range usernames {
		user, _ := db.Auth.GetUser(username)

		// Format user info
		info := "user " + username
		if user.Enabled {
			info += " on"
		} else {
			info += " off"
		}

		if user.NoPass {
			info += " nopass"
		} else {
			info += " >****" // Hide actual password
		}

		info += " ~* &* +@all" // Simplified permissions display

		result[i] = []byte(info)
	}

	return reply.MakeMultiBulkReply(result)
}

// ACL DELUSER command implementation
func AclDelUser(db *DB, args [][]byte) redis.Reply {
	if len(args) == 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'acl deluser' command")
	}

	deleted := 0
	for _, arg := range args {
		username := string(arg)
		if username == "default" {
			continue // Cannot delete default user
		}

		if db.Auth.DeleteUser(username) {
			deleted++
		}
	}

	return reply.MakeIntReply(int64(deleted))
}

// ACL command dispatcher
func AclCommand(db *DB, args [][]byte) redis.Reply {
	if len(args) == 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'acl' command")
	}

	subCommand := strings.ToLower(string(args[0]))

	switch subCommand {
	case "user":
		return AclUser(db, args[1:])
	case "list":
		return AclList(db, args[1:])
	case "deluser":
		return AclDelUser(db, args[1:])
	case "save":
		return AclSave(db, args[1:])
	case "load":
		return AclLoad(db, args[1:])
	default:
		return reply.MakeErrReply("ERR unknown subcommand '" + subCommand + "' for 'acl' command")
	}
}
