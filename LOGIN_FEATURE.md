# GoRedis Authentication Feature

This document describes the authentication and authorization features implemented in GoRedis.

## Overview

The login feature provides Redis-compatible authentication with user management, similar to Redis 6+ ACL system. It includes:

- User creation and management
- Password-based authentication  
- Command and key access control
- Session management
- **Configuration file support (NEW!)**
- **ACL file persistence (NEW!)**
- **Runtime configuration reloading (NEW!)**

## Default Users

### Admin User
- **Username**: `admin`
- **Password**: `admin123`
- **Permissions**: Full access to all commands and keys

### Default User
- **Username**: `default`
- **Password**: None (no authentication required)
- **Permissions**: Full access (can be restricted via ACL)

## Commands

### AUTH Command

Authenticate a user session.

```redis
# Authenticate with username and password
AUTH admin admin123

# Authenticate with password only (uses default user)
AUTH somepassword
```

### ACL Commands

#### ACL USER - Create/Modify Users

```redis
# Create a new user
ACL USER newuser on >password123

# Enable/disable user
ACL USER username on   # Enable user
ACL USER username off  # Disable user

# Set password
ACL USER username >newpassword

# Allow no password authentication
ACL USER username nopass
```

#### ACL LIST - List All Users

```redis
ACL LIST
```

Returns a list of all users with their current settings.

#### ACL DELUSER - Delete Users

```redis
# Delete one user
ACL DELUSER username

# Delete multiple users
ACL DELUSER user1 user2 user3
```

#### ACL SAVE - Save Users to File

```redis
ACL SAVE
```

Saves the current user configuration to the ACL file specified in `redis.conf`.

#### ACL LOAD - Reload Users from File

```redis
ACL LOAD
```

Reloads users from the ACL file, replacing current in-memory users.

## Configuration Files

### Redis Configuration File (`redis.conf`)

```bash
# Server Configuration
port 6399
databases 16

# Security - set default password
requirepass mypassword123

# ACL file path
aclfile ./config/users.acl
```

### ACL Configuration File (`users.acl`)

```acl
# Default user - no authentication required
user default on nopass ~* +@all

# Admin user - full access with password
user admin on >admin123 ~* +@all

# App user - limited to app keys only
user app_user on >app_password ~app:* +@read +@write -@dangerous

# Read-only user
user readonly on >readonly123 ~* +@read -@write -@admin
```

## Usage Examples

### 1. Basic Authentication

```bash
# Connect to GoRedis
redis-cli -p 6399

# Authenticate as admin
127.0.0.1:6399> AUTH admin admin123
OK

# Now you can execute commands
127.0.0.1:6399> SET mykey "Hello World"
OK
```

### 2. Create a New User

```bash
# Create a new user with password
127.0.0.1:6399> ACL USER bob on >bobpassword
OK

# Authenticate as the new user
127.0.0.1:6399> AUTH bob bobpassword
OK
```

### 3. User Management

```bash
# List all users
127.0.0.1:6399> ACL LIST
1) "user admin on >**** ~* &* +@all"
2) "user default on nopass ~* &* +@all"
3) "user bob on >**** ~* &* +@all"

# Delete a user
127.0.0.1:6399> ACL DELUSER bob
(integer) 1
```

## Implementation Details

### Authentication Flow

1. **Client Connection**: New clients start with the `default` user
2. **AUTH Command**: Validates credentials and updates client session
3. **Command Execution**: Each command checks user permissions
4. **Session State**: Client maintains authenticated user throughout connection

### Password Security

- Passwords are hashed using SHA-256
- Original passwords are never stored
- Hash comparison for authentication

### Permission System

The system supports:

- **Command Access Control**: Restrict which commands users can execute
- **Key Pattern Matching**: Control access to specific key patterns
- **User Enable/Disable**: Temporarily disable users without deletion

### Data Structures

```go
type User struct {
    Username     string
    PasswordHash string
    Enabled      bool
    NoPass       bool              // No password required
    Commands     map[string]bool   // Allowed commands
    Keys         []string          // Key patterns
    CreatedAt    time.Time
    LastLogin    *time.Time
}
```

## Security Features

1. **Authentication Required**: Commands can require authentication
2. **Permission Checking**: Validates command and key access
3. **Session Management**: Tracks authenticated state per connection
4. **Secure Password Storage**: Uses cryptographic hashing

## Error Messages

- `WRONGPASS invalid username-password pair`: Invalid credentials
- `NOAUTH Authentication required`: Command requires authentication  
- `NOPERM this user has no permissions to run the 'cmd' command`: Insufficient command permissions
- `NOPERM this user has no permissions to access key 'key'`: Insufficient key permissions

## Testing

Run the authentication tests:

```bash
cd src/db
go test -v -run TestAuth
```

## Integration with Redis Clients

The authentication system is compatible with standard Redis clients:

```python
import redis

# Python redis client
r = redis.Redis(host='localhost', port=6399, password='admin123', username='admin')
r.set('key', 'value')
```

```javascript
// Node.js redis client
const redis = require('redis');
const client = redis.createClient({
    host: 'localhost',
    port: 6399,
    username: 'admin',
    password: 'admin123'
});
```

## Future Enhancements

Potential improvements for the authentication system:

1. **Role-Based Access Control**: User roles with inherited permissions
2. **Key Pattern Glob Matching**: More sophisticated key pattern matching
3. **Session Timeouts**: Automatic session expiration
4. **Audit Logging**: Track user actions and authentication attempts
5. **Password Policies**: Enforce password complexity requirements
6. **Multi-Factor Authentication**: Additional security layers

## Compatibility

This implementation provides compatibility with:
- Redis 6+ ACL commands
- Standard Redis authentication patterns
- Most Redis clients that support username/password authentication

The system maintains backward compatibility with Redis clients expecting simple password authentication.