package store

import (
	"SuperChat/internal/model"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

func (s *BoltStore) CreateUser(user *model.User) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}

		usernameKey := []byte("username:" + user.Username)
		if bucket.Get(usernameKey) != nil {
			return ErrUsernameExists
		}

		userData, err := user.ToJSON()
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(user.ID), userData); err != nil {
			return err
		}
		return bucket.Put(usernameKey, []byte(user.ID))
	})
}

func (s *BoltStore) GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}

		userID := bucket.Get([]byte("username:" + username))
		if userID == nil {
			return fmt.Errorf("user not found")
		}
		userData := bucket.Get(userID)
		if userData == nil {
			return fmt.Errorf("user not found")
		}
		return json.Unmarshal(userData, user)
	})
	return user, err
}
