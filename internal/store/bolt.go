package store

import (
	"errors"

	"go.etcd.io/bbolt"
)

var ErrUsernameExists = errors.New("username already exists")
var ErrTeamNotFound = errors.New("team not found")
var ErrUserNotFound = errors.New("user not found")

type BoltStore struct {
	db *bbolt.DB
}

func NewBoltStore(db *bbolt.DB) *BoltStore {
	return &BoltStore{db: db}
}
