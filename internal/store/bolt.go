package store

import (
	"errors"

	"go.etcd.io/bbolt"
)

var ErrUsernameExists = errors.New("username already exists")

type BoltStore struct {
	db *bbolt.DB
}

func NewBoltStore(db *bbolt.DB) *BoltStore {
	return &BoltStore{db: db}
}
