package store

import (
	"SuperChat/internal/model"
	"encoding/json"
	"fmt"

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
