package db

import (
	"encoding/json"
	"time"
)

const AUDIT_LOG_ENTRY_SCHEMA = `
CREATE TABLE IF NOT EXISTS audit_log_entries (
	id INTEGER PRIMARY KEY,
	action TEXT,
	user_id INTEGER,
	created_at INTEGER,
	data TEXT
);
`

type AuditLogEntry struct {
	Id        int64  `json:"id" db:"id"`
	Action    string `json:"action" db:"action"`
	UserId    int64  `json:"user_id" db:"user_id"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
	RawData   string `json:"-" db:"data"`

	Data map[string]interface{} `json:"data" db:"-"`
}

func CreateAuditLogEntry(action string, user *User, data map[string]interface{}) (*AuditLogEntry, error) {
	dataEncoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ts := time.Now().Unix()

	result, err := db.Exec(
		`INSERT INTO audit_log_entries (action, user_id, created_at, data) VALUES (?, ?, ?, ?);`,
		action,
		user.Id,
		ts,
		dataEncoded,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &AuditLogEntry{
		Id:        id,
		Action:    action,
		UserId:    user.Id,
		CreatedAt: ts,
		Data:      data,
	}, nil
}

func GetRecentAuditLogEntries(limit int) ([]AuditLogEntry, error) {
	var entries []AuditLogEntry
	err := db.Select(&entries, `SELECT * FROM audit_log_entries ORDER BY action DESC LIMIT ?`, limit)
	if err != nil {
		return make([]AuditLogEntry, 0), err
	}

	return entries, err
}
