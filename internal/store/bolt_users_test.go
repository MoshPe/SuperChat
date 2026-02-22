package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"SuperChat/internal/model"
	"go.etcd.io/bbolt"
)

func withTempBoltStore(t *testing.T, fn func(*BoltStore)) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "store-test.db")
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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

	fn(NewBoltStore(db))
}

func TestBoltStoreCreateUserAndGetUserByUsername(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		u := model.NewUser("alice", "Alice", "hashed")

		if err := s.CreateUser(u); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		got, err := s.GetUserByUsername("alice")
		if err != nil {
			t.Fatalf("GetUserByUsername: %v", err)
		}
		if got.ID != u.ID || got.Username != "alice" || got.Name != "Alice" {
			t.Fatalf("unexpected user: %#v", got)
		}
	})
}

func TestBoltStoreCreateUserRejectsDuplicateUsername(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		if err := s.CreateUser(model.NewUser("alice", "Alice", "hashed")); err != nil {
			t.Fatalf("first CreateUser: %v", err)
		}

		err := s.CreateUser(model.NewUser("alice", "Alice 2", "hashed2"))
		if !errors.Is(err, ErrUsernameExists) {
			t.Fatalf("expected ErrUsernameExists, got %v", err)
		}
	})
}
