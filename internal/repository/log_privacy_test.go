package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDatabaseErrorsDoNotLogSensitiveParameters(t *testing.T) {
	const marker = "private-session-token-must-not-appear"
	if os.Getenv("TEST_USER_DB_LOG_PRIVACY") == "1" {
		db, err := InitDB(filepath.Join(t.TempDir(), "privacy.db"))
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()
		var rows []map[string]interface{}
		if err := db.Table("missing_private_table").Where("token = ?", marker).Find(&rows).Error; err == nil {
			t.Fatal("test query unexpectedly succeeded")
		}
		return
	}
	// Capture writers created before the test starts as well as the current logger.
	cmd := exec.Command(os.Args[0], "-test.run=^TestDatabaseErrorsDoNotLogSensitiveParameters$")
	cmd.Env = append(os.Environ(), "TEST_USER_DB_LOG_PRIVACY=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("logging probe failed: %v", err)
	}
	if strings.Contains(string(output), marker) {
		t.Fatal("SQL logger leaked a session token")
	}
	if !strings.Contains(string(output), "missing_private_table") {
		t.Fatal("database error was not logged")
	}
}
