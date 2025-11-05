package db

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/geoffreyhinton/goredis/src/interface/redis"
	"github.com/geoffreyhinton/goredis/src/lib/logger"
	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

// Config represents the Redis configuration
type Config struct {
	RequirePass string `json:"requirepass"`
	ACLFile     string `json:"aclfile"`
	Port        int    `json:"port"`
	Database    int    `json:"databases"`
}

// LoadConfigFromFile loads configuration from a redis.conf style file
func LoadConfigFromFile(configPath string) (*Config, error) {
	config := &Config{
		Port:     6399,
		Database: 16,
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Info("Config file not found, using defaults: " + configPath)
		return config, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "requirepass":
			config.RequirePass = value
		case "aclfile":
			config.ACLFile = value
		case "port":
			if port, err := strconv.Atoi(value); err == nil {
				config.Port = port
			}
		case "databases":
			if db, err := strconv.Atoi(value); err == nil {
				config.Database = db
			}
		}
	}

	return config, scanner.Err()
}

// LoadUsersFromACLFile loads users from an ACL file
func (am *AuthManager) LoadUsersFromACLFile(aclFile string) error {
	if aclFile == "" {
		return nil
	}

	if _, err := os.Stat(aclFile); os.IsNotExist(err) {
		logger.Info("ACL file not found, creating with defaults: " + aclFile)
		return am.SaveUsersToACLFile(aclFile)
	}

	file, err := os.Open(aclFile)
	if err != nil {
		return fmt.Errorf("error opening ACL file: %v", err)
	}
	defer file.Close()

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Clear existing users except default
	am.users = make(map[string]*User)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := am.parseACLLine(line); err != nil {
			logger.Warn(fmt.Sprintf("Error parsing ACL line %d: %v", lineNum, err))
		}
	}

	logger.Info(fmt.Sprintf("Loaded %d users from ACL file: %s", len(am.users), aclFile))
	return scanner.Err()
}

// parseACLLine parses a single ACL line from the file
func (am *AuthManager) parseACLLine(line string) error {
	parts := strings.Fields(line)
	if len(parts) < 2 || strings.ToLower(parts[0]) != "user" {
		return fmt.Errorf("invalid ACL line format: %s", line)
	}

	username := parts[1]
	user := &User{
		Username:  username,
		Enabled:   false,
		NoPass:    false,
		Commands:  make(map[string]bool),
		Keys:      []string{},
		CreatedAt: time.Now(),
		Flags:     make(map[string]bool),
	}

	// Parse user attributes
	for i := 2; i < len(parts); i++ {
		attr := parts[i]

		switch {
		case attr == "on":
			user.Enabled = true
		case attr == "off":
			user.Enabled = false
		case attr == "nopass":
			user.NoPass = true
		case strings.HasPrefix(attr, ">"):
			// Password
			password := attr[1:]
			user.PasswordHash = hashPassword(password)
			user.NoPass = false
		case strings.HasPrefix(attr, "~"):
			// Key pattern
			pattern := attr[1:]
			user.Keys = append(user.Keys, pattern)
		case strings.HasPrefix(attr, "+"):
			// Allow command/category
			cmd := strings.ToLower(attr[1:])
			if cmd == "@all" {
				user.Commands = nil // nil means all commands allowed
			} else {
				if user.Commands == nil {
					user.Commands = make(map[string]bool)
				}
				user.Commands[cmd] = true
			}
		case strings.HasPrefix(attr, "-"):
			// Deny command/category
			cmd := strings.ToLower(attr[1:])
			if user.Commands == nil {
				user.Commands = make(map[string]bool)
			}
			user.Commands[cmd] = false
		}
	}

	// Default to all keys if none specified
	if len(user.Keys) == 0 {
		user.Keys = []string{"*"}
	}

	am.users[username] = user
	return nil
}

// SaveUsersToACLFile saves current users to an ACL file
func (am *AuthManager) SaveUsersToACLFile(aclFile string) error {
	if aclFile == "" {
		return fmt.Errorf("no ACL file specified")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(aclFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating ACL directory: %v", err)
	}

	am.mutex.RLock()
	defer am.mutex.RUnlock()

	content := fmt.Sprintf("# Redis ACL file generated on %s\n", time.Now().Format(time.RFC3339))
	content += "# Format: user <username> <attributes>\n\n"

	// Add default user
	content += am.formatUserACL(am.defaultUser) + "\n"

	// Add all other users
	for _, user := range am.users {
		content += am.formatUserACL(user) + "\n"
	}

	if err := ioutil.WriteFile(aclFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing ACL file: %v", err)
	}

	logger.Info(fmt.Sprintf("Saved %d users to ACL file: %s", len(am.users)+1, aclFile))
	return nil
}

// formatUserACL formats a user as an ACL line
func (am *AuthManager) formatUserACL(user *User) string {
	line := fmt.Sprintf("user %s", user.Username)

	if user.Enabled {
		line += " on"
	} else {
		line += " off"
	}

	if user.NoPass {
		line += " nopass"
	} else if user.PasswordHash != "" {
		line += " >****" // Don't expose actual password in saved file
	}

	// Key patterns
	for _, pattern := range user.Keys {
		line += fmt.Sprintf(" ~%s", pattern)
	}

	// Commands
	if user.Commands == nil {
		line += " +@all"
	} else {
		for cmd, allowed := range user.Commands {
			if allowed {
				line += fmt.Sprintf(" +%s", cmd)
			} else {
				line += fmt.Sprintf(" -%s", cmd)
			}
		}
	}

	return line
}

// LoadConfigFromJSON loads configuration from a JSON file (alternative format)
func LoadConfigFromJSON(configPath string) (*Config, error) {
	config := &Config{
		Port:     6399,
		Database: 16,
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing JSON config: %v", err)
	}

	return config, nil
}

// ACL SAVE command implementation
func AclSave(db *DB, args [][]byte) redis.Reply {
	if len(args) != 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'acl save' command")
	}

	aclFile := db.Config.ACLFile
	if aclFile == "" {
		return reply.MakeErrReply("ERR The server is running without an ACL config file")
	}

	if err := db.Auth.SaveUsersToACLFile(aclFile); err != nil {
		return reply.MakeErrReply(fmt.Sprintf("ERR Failed to save ACL file: %v", err))
	}

	return reply.MakeStatusReply("OK")
}

// ACL LOAD command implementation
func AclLoad(db *DB, args [][]byte) redis.Reply {
	if len(args) != 0 {
		return reply.MakeErrReply("ERR wrong number of arguments for 'acl load' command")
	}

	aclFile := db.Config.ACLFile
	if aclFile == "" {
		return reply.MakeErrReply("ERR The server is running without an ACL config file")
	}

	if err := db.Auth.LoadUsersFromACLFile(aclFile); err != nil {
		return reply.MakeErrReply(fmt.Sprintf("ERR Failed to load ACL file: %v", err))
	}

	return reply.MakeStatusReply("OK")
}
