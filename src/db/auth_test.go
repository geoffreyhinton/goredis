package db

import (
	"testing"

	"github.com/geoffreyhinton/goredis/src/redis/reply"
)

func TestAuthManager_CreateUser(t *testing.T) {
	auth := NewAuthManager()

	// Test creating a new user
	err := auth.CreateUser("testuser", "password123")
	if err != nil {
		t.Errorf("CreateUser failed: %v", err)
	}

	// Test creating duplicate user
	err = auth.CreateUser("testuser", "password456")
	if err == nil {
		t.Error("Expected error when creating duplicate user")
	}

	// Verify user exists
	user, exists := auth.GetUser("testuser")
	if !exists {
		t.Error("User should exist after creation")
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}
}

func TestAuthManager_Authenticate(t *testing.T) {
	auth := NewAuthManager()

	// Test authenticating admin user
	user, err := auth.Authenticate("admin", "admin123")
	if err != nil {
		t.Errorf("Admin authentication failed: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", user.Username)
	}
	if user.LastLogin == nil {
		t.Error("LastLogin should be set after authentication")
	}

	// Test wrong password
	_, err = auth.Authenticate("admin", "wrongpassword")
	if err == nil {
		t.Error("Expected error with wrong password")
	}

	// Test non-existent user
	_, err = auth.Authenticate("nonexistent", "password")
	if err == nil {
		t.Error("Expected error with non-existent user")
	}
}

func TestAuth_Command(t *testing.T) {
	db := MakeDB()

	// Test AUTH with username and password
	result := Auth(db, [][]byte{[]byte("admin"), []byte("admin123")})
	if _, ok := result.(*reply.OkReply); !ok {
		t.Error("AUTH command should succeed with correct credentials")
	}

	// Test AUTH with wrong password
	result = Auth(db, [][]byte{[]byte("admin"), []byte("wrongpassword")})
	if _, ok := result.(*reply.ErrReply); !ok {
		t.Error("AUTH command should fail with wrong password")
	}

	// Test AUTH with wrong number of arguments
	result = Auth(db, [][]byte{})
	if _, ok := result.(*reply.ErrReply); !ok {
		t.Error("AUTH command should fail with no arguments")
	}
}

func TestAcl_Commands(t *testing.T) {
	db := MakeDB()

	// Test ACL USER command to create a new user
	result := AclUser(db, [][]byte{[]byte("newuser"), []byte("on"), []byte(">newpassword")})
	if _, ok := result.(*reply.OkReply); !ok {
		t.Error("ACL USER command should succeed")
	}

	// Test ACL LIST command
	result = AclList(db, [][]byte{})
	if multiBulk, ok := result.(*reply.MultiBulkReply); ok {
		if len(multiBulk.Args) == 0 {
			t.Error("ACL LIST should return users")
		}
	} else {
		t.Error("ACL LIST should return MultiBulkReply")
	}

	// Test ACL DELUSER command
	result = AclDelUser(db, [][]byte{[]byte("newuser")})
	if intReply, ok := result.(*reply.IntReply); ok {
		if intReply.Code != 1 {
			t.Errorf("Expected 1 user deleted, got %d", intReply.Code)
		}
	} else {
		t.Error("ACL DELUSER should return IntReply")
	}
}

func TestUser_Permissions(t *testing.T) {
	user := &User{
		Username: "testuser",
		Enabled:  true,
		Commands: map[string]bool{
			"get": true,
			"set": true,
		},
		Keys: []string{"user:*", "session:123"},
	}

	// Test command permissions
	if !user.CanExecuteCommand("get") {
		t.Error("User should be able to execute GET command")
	}
	if user.CanExecuteCommand("del") {
		t.Error("User should not be able to execute DEL command")
	}

	// Test key permissions
	if !user.CanAccessKey("user:123") {
		t.Error("User should be able to access user:123 key")
	}
	if !user.CanAccessKey("session:123") {
		t.Error("User should be able to access session:123 key")
	}
	if user.CanAccessKey("admin:config") {
		t.Error("User should not be able to access admin:config key")
	}
}

func TestAuthenticatedExec(t *testing.T) {
	db := MakeDB()
	ctx := db.CreateAuthContext()

	// Test command execution without authentication (should work with default user)
	result := db.AuthenticatedExec(ctx, [][]byte{[]byte("ping")})
	if _, ok := result.(*reply.PongReply); !ok {
		t.Error("PING should work without authentication")
	}

	// Test AUTH command
	result = db.AuthenticatedExec(ctx, [][]byte{[]byte("auth"), []byte("admin"), []byte("admin123")})
	if _, ok := result.(*reply.OkReply); !ok {
		t.Error("AUTH should succeed with correct credentials")
	}

	// After AUTH, user should be updated
	if !ctx.Authenticated {
		t.Error("Context should be authenticated after successful AUTH")
	}
	if ctx.User.Username != "admin" {
		t.Errorf("Expected user 'admin', got '%s'", ctx.User.Username)
	}
}
