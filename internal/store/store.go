package store

import "SuperChat/internal/model"

type UserStore interface {
	CreateUser(user *model.User) error
	GetUserByUsername(username string) (*model.User, error)
}
