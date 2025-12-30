package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

var errUsernameExists = errors.New("username already exists")

// Database operations for users
func createUser(user *User) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}

		// Check if username already exists
		usernameKey := []byte("username:" + user.Username)
		if bucket.Get(usernameKey) != nil {
			return errUsernameExists
		}

		// Store user by ID
		userData, err := user.ToJSON()
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(user.ID), userData); err != nil {
			return err
		}

		// Store user by username for lookup
		if err := bucket.Put(usernameKey, []byte(user.ID)); err != nil {
			return err
		}

		return nil
	})
}

func getUserByID(userID string) (*User, error) {
	var user *User
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}

		userData := bucket.Get([]byte(userID))
		if userData == nil {
			return fmt.Errorf("user not found")
		}

		var u User
		if err := json.Unmarshal(userData, &u); err != nil {
			return err
		}
		user = &u
		return nil
	})

	return user, err
}

func getUserByUsername(username string) (*User, error) {
	user := &User{} // allocate on heap
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}

		// Lookup user ID by username
		usernameKey := []byte("username:" + username)
		userID := bucket.Get(usernameKey)
		if userID == nil {
			return fmt.Errorf("user not found")
		}

		// Lookup full user data by ID
		userData := bucket.Get(userID)
		if userData == nil {
			return fmt.Errorf("user not found")
		}

		if err := json.Unmarshal(userData, user); err != nil {
			return err
		}
		return nil
	})

	return user, err
}

// getAllUsers returns all users
func getAllUsers() ([]*User, error) {
	var users []*User
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			// skip username index keys (prefix "username:")
			if len(k) >= 9 && string(k[:9]) == "username:" {
				continue
			}
			var u User
			if err := json.Unmarshal(v, &u); err != nil {
				continue
			}
			users = append(users, &u)
		}
		return nil
	})
	return users, err
}

func updateUser(user *User) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return fmt.Errorf("users bucket not found")
		}

		userData, err := user.ToJSON()
		if err != nil {
			return err
		}

		return bucket.Put([]byte(user.ID), userData)
	})
}

// Database operations for teams
func createTeam(team *Team) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("teams"))
		if bucket == nil {
			return fmt.Errorf("teams bucket not found")
		}

		teamData, err := team.ToJSON()
		if err != nil {
			return err
		}

		return bucket.Put([]byte(team.ID), teamData)
	})
}

func getTeamByID(teamID string) (*Team, error) {
	var team *Team
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("teams"))
		if bucket == nil {
			return fmt.Errorf("teams bucket not found")
		}

		teamData := bucket.Get([]byte(teamID))
		if teamData == nil {
			return fmt.Errorf("team not found")
		}

		var t Team
		if err := json.Unmarshal(teamData, &t); err != nil {
			return err
		}
		team = &t
		return nil
	})

	return team, err
}

func getUserTeams(userID string) ([]*Team, error) {
	var teams []*Team
	err := db.View(func(tx *bbolt.Tx) error {
		teamsBucket := tx.Bucket([]byte("teams"))
		membersBucket := tx.Bucket([]byte("team_members"))
		if teamsBucket == nil || membersBucket == nil {
			return fmt.Errorf("required buckets not found")
		}

		// Get all team memberships for the user
		c := membersBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}

			if member.UserID == userID {
				// Get team data
				teamData := teamsBucket.Get([]byte(member.TeamID))
				if teamData != nil {
					var team Team
					if err := json.Unmarshal(teamData, &team); err != nil {
						continue
					}
					teams = append(teams, &team)
				}
			}
		}

		return nil
	})

	return teams, err
}

// Database operations for team members
func addTeamMember(member *TeamMember) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}

		memberData, err := member.ToJSON()
		if err != nil {
			return err
		}

		return bucket.Put([]byte(member.ID), memberData)
	})
}

func removeTeamMember(teamID, userID string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}

		// Find and remove the membership
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}

			if member.TeamID == teamID && member.UserID == userID {
				return bucket.Delete(k)
			}
		}

		return fmt.Errorf("team membership not found")
	})
}

// getTeamMembers returns all members for a team
func getTeamMembers(teamID string) ([]*TeamMember, error) {
	var members []*TeamMember
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}
			if member.TeamID == teamID {
				// capture member copy
				m := member
				members = append(members, &m)
			}
		}
		return nil
	})
	return members, err
}

func isTeamMember(teamID, userID string) bool {
	var isMember bool
	db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}

			if member.TeamID == teamID && member.UserID == userID {
				isMember = true
				break
			}
		}

		return nil
	})

	return isMember
}

// Database operations for messages
func saveMessage(message *Message) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("messages"))
		if bucket == nil {
			return fmt.Errorf("messages bucket not found")
		}

		messageData, err := message.ToJSON()
		if err != nil {
			return err
		}

		return bucket.Put([]byte(message.ID), messageData)
	})
}

func getTeamMessages(teamID string, limit int) ([]*Message, error) {
	var messages []*Message
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("messages"))
		if bucket == nil {
			return fmt.Errorf("messages bucket not found")
		}

		c := bucket.Cursor()
		// Start from the end to get most recent messages
		for k, v := c.Last(); k != nil && len(messages) < limit; k, v = c.Prev() {
			var message Message
			if err := json.Unmarshal(v, &message); err != nil {
				continue
			}

			// TTL filter: include only messages within the last 7 days
			if message.TeamID == teamID && message.CreatedAt.After(time.Now().Add(-7*24*time.Hour)) {
				messages = append([]*Message{&message}, messages...) // Prepend to maintain order
			}
		}

		return nil
	})

	return messages, err
}

// getMessageByID returns a single message by ID
func getMessageByID(id string) (*Message, error) {
	var message *Message
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("messages"))
		if bucket == nil {
			return fmt.Errorf("messages bucket not found")
		}
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("message not found")
		}
		var m Message
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		message = &m
		return nil
	})
	return message, err
}

// deleteMessage deletes a message by ID
func deleteMessage(id string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("messages"))
		if bucket == nil {
			return fmt.Errorf("messages bucket not found")
		}
		return bucket.Delete([]byte(id))
	})
}

// deleteExpiredMessages removes messages older than the provided ttl
func deleteExpiredMessages(ttl time.Duration) error {
	cutoff := time.Now().Add(-ttl)
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("messages"))
		if bucket == nil {
			return fmt.Errorf("messages bucket not found")
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var message Message
			if err := json.Unmarshal(v, &message); err != nil {
				continue
			}
			if message.CreatedAt.Before(cutoff) {
				if err := bucket.Delete(k); err != nil {
					// keep going even if a single delete fails
					continue
				}
			}
		}
		return nil
	})
}

// startMessageTTLJanitor periodically deletes expired messages
func startMessageTTLJanitor(ttl time.Duration, interval time.Duration) {
	// Run once at startup
	_ = deleteExpiredMessages(ttl)
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			_ = deleteExpiredMessages(ttl)
		}
	}()
}

// Upload storage in BoltDB
func saveUpload(filename, contentType, ownerID, teamID string, data []byte) (string, error) {
	id := uuid.New().String()
	now := time.Now()
	meta := &UploadMeta{ID: id, Filename: filename, ContentType: contentType, OwnerID: ownerID, TeamID: teamID, CreatedAt: now}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		metaB := tx.Bucket([]byte("uploads"))
		dataB := tx.Bucket([]byte("upload_data"))
		if metaB == nil || dataB == nil {
			return fmt.Errorf("upload buckets not found")
		}
		if err := metaB.Put([]byte(id), metaBytes); err != nil {
			return err
		}
		if err := dataB.Put([]byte(id), data); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func getUpload(id string) (*UploadMeta, []byte, error) {
	var meta UploadMeta
	var data []byte
	err := db.View(func(tx *bbolt.Tx) error {
		metaB := tx.Bucket([]byte("uploads"))
		dataB := tx.Bucket([]byte("upload_data"))
		if metaB == nil || dataB == nil {
			return fmt.Errorf("upload buckets not found")
		}
		m := metaB.Get([]byte(id))
		if m == nil {
			return fmt.Errorf("not found")
		}
		if err := json.Unmarshal(m, &meta); err != nil {
			return err
		}
		d := dataB.Get([]byte(id))
		if d == nil {
			return fmt.Errorf("not found")
		}
		// Copy bytes out of Bolt buffer
		data = append([]byte(nil), d...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &meta, data, nil
}

func deleteExpiredUploads(ttl time.Duration) error {
	cutoff := time.Now().Add(-ttl)
	return db.Update(func(tx *bbolt.Tx) error {
		metaB := tx.Bucket([]byte("uploads"))
		dataB := tx.Bucket([]byte("upload_data"))
		if metaB == nil || dataB == nil {
			return fmt.Errorf("upload buckets not found")
		}
		c := metaB.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var meta UploadMeta
			if err := json.Unmarshal(v, &meta); err != nil {
				continue
			}
			if meta.CreatedAt.Before(cutoff) {
				_ = metaB.Delete(k)
				_ = dataB.Delete(k)
			}
		}
		return nil
	})
}

func startUploadTTLJanitor(ttl time.Duration, interval time.Duration) {
	_ = deleteExpiredUploads(ttl)
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			_ = deleteExpiredUploads(ttl)
		}
	}()
}

// Helper function to check if a user owns a team
func isTeamOwner(teamID, userID string) bool {
	team, err := getTeamByID(teamID)
	if err != nil {
		return false
	}
	return team.OwnerID == userID
}
