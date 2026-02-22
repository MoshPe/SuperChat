package store

import (
	"SuperChat/internal/model"
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

func (s *BoltStore) CreateTeam(team *model.Team) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
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

func (s *BoltStore) GetTeamByID(teamID string) (*model.Team, error) {
	var team *model.Team
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("teams"))
		if bucket == nil {
			return fmt.Errorf("teams bucket not found")
		}

		teamData := bucket.Get([]byte(teamID))
		if teamData == nil {
			return ErrTeamNotFound
		}

		var t model.Team
		if err := json.Unmarshal(teamData, &t); err != nil {
			return err
		}
		team = &t
		return nil
	})
	return team, err
}

func (s *BoltStore) GetUserTeams(userID string) ([]*model.Team, error) {
	var teams []*model.Team
	err := s.db.View(func(tx *bbolt.Tx) error {
		teamsBucket := tx.Bucket([]byte("teams"))
		membersBucket := tx.Bucket([]byte("team_members"))
		if teamsBucket == nil || membersBucket == nil {
			return fmt.Errorf("required buckets not found")
		}

		c := membersBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member model.TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}
			if member.UserID != userID {
				continue
			}

			teamData := teamsBucket.Get([]byte(member.TeamID))
			if teamData == nil {
				continue
			}
			var team model.Team
			if err := json.Unmarshal(teamData, &team); err != nil {
				continue
			}
			t := team
			teams = append(teams, &t)
		}

		return nil
	})
	return teams, err
}

func (s *BoltStore) UpdateTeam(team *model.Team) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
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

func (s *BoltStore) DeleteTeam(teamID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		teamsB := tx.Bucket([]byte("teams"))
		membersB := tx.Bucket([]byte("team_members"))
		messagesB := tx.Bucket([]byte("messages"))
		uploadsB := tx.Bucket([]byte("uploads"))
		uploadDataB := tx.Bucket([]byte("upload_data"))
		if teamsB == nil || membersB == nil || messagesB == nil || uploadsB == nil || uploadDataB == nil {
			return fmt.Errorf("required buckets not found")
		}

		if err := teamsB.Delete([]byte(teamID)); err != nil {
			return err
		}

		mc := membersB.Cursor()
		for k, v := mc.First(); k != nil; k, v = mc.Next() {
			var member model.TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}
			if member.TeamID == teamID {
				_ = membersB.Delete(k)
			}
		}

		msgC := messagesB.Cursor()
		for k, v := msgC.First(); k != nil; k, v = msgC.Next() {
			var m model.Message
			if err := json.Unmarshal(v, &m); err != nil {
				continue
			}
			if m.TeamID == teamID {
				_ = messagesB.Delete(k)
			}
		}

		upC := uploadsB.Cursor()
		for k, v := upC.First(); k != nil; k, v = upC.Next() {
			var meta model.UploadMeta
			if err := json.Unmarshal(v, &meta); err != nil {
				continue
			}
			if meta.TeamID == teamID {
				_ = uploadsB.Delete(k)
				_ = uploadDataB.Delete(k)
			}
		}
		return nil
	})
}

func (s *BoltStore) AddTeamMember(member *model.TeamMember) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
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

func (s *BoltStore) RemoveTeamMember(teamID, userID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member model.TeamMember
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

func (s *BoltStore) GetTeamMembers(teamID string) ([]*model.TeamMember, error) {
	var members []*model.TeamMember
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member model.TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}
			if member.TeamID == teamID {
				m := member
				members = append(members, &m)
			}
		}
		return nil
	})
	return members, err
}

func (s *BoltStore) IsTeamMember(teamID, userID string) (bool, error) {
	var isMember bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("team_members"))
		if bucket == nil {
			return fmt.Errorf("team_members bucket not found")
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member model.TeamMember
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
	return isMember, err
}

func (s *BoltStore) TransferTeamOwnership(teamID, currentOwnerID, newOwnerID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		teamsBucket := tx.Bucket([]byte("teams"))
		membersBucket := tx.Bucket([]byte("team_members"))
		if teamsBucket == nil || membersBucket == nil {
			return fmt.Errorf("required buckets not found")
		}

		teamData := teamsBucket.Get([]byte(teamID))
		if teamData == nil {
			return ErrTeamNotFound
		}

		var team model.Team
		if err := json.Unmarshal(teamData, &team); err != nil {
			return err
		}
		if team.OwnerID != currentOwnerID {
			return fmt.Errorf("forbidden")
		}
		if newOwnerID == currentOwnerID {
			return fmt.Errorf("cannot transfer ownership to yourself")
		}

		var newOwnerMemberKey []byte
		var oldOwnerMemberKey []byte
		c := membersBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var member model.TeamMember
			if err := json.Unmarshal(v, &member); err != nil {
				continue
			}
			if member.TeamID != teamID {
				continue
			}
			if member.UserID == newOwnerID {
				newOwnerMemberKey = append([]byte(nil), k...)
			}
			if member.UserID == currentOwnerID {
				oldOwnerMemberKey = append([]byte(nil), k...)
			}
		}
		if newOwnerMemberKey == nil {
			return fmt.Errorf("new owner must be an existing team member")
		}

		team.OwnerID = newOwnerID
		team.UpdatedAt = time.Now()
		updatedTeamData, err := team.ToJSON()
		if err != nil {
			return err
		}
		if err := teamsBucket.Put([]byte(team.ID), updatedTeamData); err != nil {
			return err
		}

		if raw := membersBucket.Get(newOwnerMemberKey); raw != nil {
			var m model.TeamMember
			if err := json.Unmarshal(raw, &m); err == nil {
				m.Role = "owner"
				if data, err := m.ToJSON(); err == nil {
					_ = membersBucket.Put(newOwnerMemberKey, data)
				}
			}
		}
		if oldOwnerMemberKey != nil && string(oldOwnerMemberKey) != string(newOwnerMemberKey) {
			if raw := membersBucket.Get(oldOwnerMemberKey); raw != nil {
				var m model.TeamMember
				if err := json.Unmarshal(raw, &m); err == nil && m.Role == "owner" {
					m.Role = "member"
					if data, err := m.ToJSON(); err == nil {
						_ = membersBucket.Put(oldOwnerMemberKey, data)
					}
				}
			}
		}
		return nil
	})
}
