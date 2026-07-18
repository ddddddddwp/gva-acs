package initialize

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openIdentifierCutoverDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func execIdentifierDDL(t *testing.T, db *gorm.DB, statement string) {
	t.Helper()
	if err := db.Exec(statement).Error; err != nil {
		t.Fatalf("exec %q: %v", statement, err)
	}
}

func TestCutoverCommandIdentifierSchemaRenamesAndPreservesValue(t *testing.T) {
	db := openIdentifierCutoverDB(t)
	execIdentifierDDL(t, db, `CREATE TABLE tr069_commands (command_id text primary key, request_id text)`)
	execIdentifierDDL(t, db, `INSERT INTO tr069_commands(command_id, request_id) VALUES ('cmd-1', 'cwmp-1')`)
	execIdentifierDDL(t, db, `CREATE TABLE tr069_command_xmls (id integer primary key, request_id text, cwmp_id text, payload blob)`)

	if err := cutoverCommandIdentifierSchema(db); err != nil {
		t.Fatalf("cutover schema: %v", err)
	}

	if db.Migrator().HasColumn("tr069_commands", "request_id") {
		t.Fatal("tr069_commands.request_id still exists")
	}
	if !db.Migrator().HasColumn("tr069_commands", "cwmp_id") {
		t.Fatal("tr069_commands.cwmp_id is missing")
	}
	var got string
	if err := db.Table("tr069_commands").Select("cwmp_id").Where("command_id = ?", "cmd-1").Scan(&got).Error; err != nil {
		t.Fatalf("read renamed value: %v", err)
	}
	if got != "cwmp-1" {
		t.Fatalf("cwmp_id = %q, want cwmp-1", got)
	}
	if db.Migrator().HasColumn("tr069_command_xmls", "request_id") {
		t.Fatal("tr069_command_xmls.request_id still exists")
	}

	if err := cutoverCommandIdentifierSchema(db); err != nil {
		t.Fatalf("repeat cutover: %v", err)
	}
}

func TestCutoverCommandIdentifierSchemaRejectsAmbiguousDualColumns(t *testing.T) {
	db := openIdentifierCutoverDB(t)
	execIdentifierDDL(t, db, `CREATE TABLE tr069_commands (command_id text primary key, request_id text, cwmp_id text)`)

	err := cutoverCommandIdentifierSchema(db)
	if err == nil || !strings.Contains(err.Error(), "both request_id and cwmp_id") {
		t.Fatalf("cutover error = %v, want ambiguous schema error", err)
	}
}

func TestCutoverCommandIdentifierSchemaAllowsFreshDatabase(t *testing.T) {
	db := openIdentifierCutoverDB(t)
	if err := cutoverCommandIdentifierSchema(db); err != nil {
		t.Fatalf("fresh database cutover: %v", err)
	}
}
