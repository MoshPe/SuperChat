package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func withTempDB(t *testing.T, fn func()) {
	t.Helper()

	origDB := db
	tmp := filepath.Join(t.TempDir(), "test.db")

	testDB, err := bbolt.Open(tmp, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}

	db = testDB
	t.Cleanup(func() {
		_ = testDB.Close()
		db = origDB
	})

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range []string{"users", "teams", "messages", "team_members", "uploads", "upload_data"} {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("init buckets: %v", err)
	}

	fn()
}

func putMessageRecord(t *testing.T, m Message) {
	t.Helper()
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		raw, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return b.Put([]byte(m.ID), raw)
	})
	if err != nil {
		t.Fatalf("put message: %v", err)
	}
}

func TestGetTeamMessagesOrdersByCreatedAtAndLimitsLatest(t *testing.T) {
	withTempDB(t, func() {
		now := time.Now()
		teamID := "team-1"

		// Keys are intentionally out of chronological order to prove key order
		// must not be used as message time order.
		putMessageRecord(t, Message{
			ID:        "z-old",
			TeamID:    teamID,
			UserID:    "u1",
			Username:  "alice",
			Content:   "old",
			Type:      "text",
			CreatedAt: now.Add(-3 * time.Minute),
		})
		putMessageRecord(t, Message{
			ID:        "a-new",
			TeamID:    teamID,
			UserID:    "u1",
			Username:  "alice",
			Content:   "new",
			Type:      "text",
			CreatedAt: now.Add(-1 * time.Minute),
		})
		putMessageRecord(t, Message{
			ID:        "m-mid",
			TeamID:    teamID,
			UserID:    "u1",
			Username:  "alice",
			Content:   "mid",
			Type:      "text",
			CreatedAt: now.Add(-2 * time.Minute),
		})

		got, err := getTeamMessages(teamID, 2)
		if err != nil {
			t.Fatalf("getTeamMessages: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(got))
		}
		if got[0].ID != "m-mid" || got[1].ID != "a-new" {
			t.Fatalf("expected latest two in chronological order [m-mid a-new], got [%s %s]", got[0].ID, got[1].ID)
		}
	})
}

func TestSortMessagesChronologicallyBreaksTimestampTiesByID(t *testing.T) {
	ts := time.Now()
	msgs := []*Message{
		{ID: "b", CreatedAt: ts},
		{ID: "a", CreatedAt: ts},
		{ID: "c", CreatedAt: ts.Add(time.Second)},
	}

	sortMessagesChronologically(msgs)

	if msgs[0].ID != "a" || msgs[1].ID != "b" || msgs[2].ID != "c" {
		t.Fatalf("expected deterministic order [a b c], got [%s %s %s]", msgs[0].ID, msgs[1].ID, msgs[2].ID)
	}
}
