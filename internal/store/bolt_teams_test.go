package store

import (
	"SuperChat/internal/model"
	"encoding/json"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func putStoreRecord(t *testing.T, db *bbolt.DB, bucket string, key []byte, value interface{}) {
	t.Helper()
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return b.Put(key, raw)
	})
	if err != nil {
		t.Fatalf("put %s record: %v", bucket, err)
	}
}

func mustBucketGet(t *testing.T, db *bbolt.DB, bucket string, key []byte) []byte {
	t.Helper()
	var out []byte
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			t.Fatalf("bucket %s missing", bucket)
		}
		if v := b.Get(key); v != nil {
			out = append([]byte(nil), v...)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("bucket get %s: %v", bucket, err)
	}
	return out
}

func TestBoltStoreTeamCRUDAndQueries(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		owner := "user-1"
		team := model.NewTeam("Team A", "desc", owner)

		if err := s.CreateTeam(team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}

		member := model.NewTeamMember(team.ID, owner, "owner")
		putStoreRecord(t, s.db, "team_members", []byte(member.ID), member)

		got, err := s.GetTeamByID(team.ID)
		if err != nil {
			t.Fatalf("GetTeamByID: %v", err)
		}
		if got.ID != team.ID || got.Name != "Team A" || got.OwnerID != owner {
			t.Fatalf("unexpected team: %#v", got)
		}

		list, err := s.GetUserTeams(owner)
		if err != nil {
			t.Fatalf("GetUserTeams: %v", err)
		}
		if len(list) != 1 || list[0].ID != team.ID {
			t.Fatalf("expected one team %s, got %#v", team.ID, list)
		}

		team.Name = "Team B"
		team.Description = "updated"
		team.UpdatedAt = time.Now()
		if err := s.UpdateTeam(team); err != nil {
			t.Fatalf("UpdateTeam: %v", err)
		}

		got, err = s.GetTeamByID(team.ID)
		if err != nil {
			t.Fatalf("GetTeamByID after update: %v", err)
		}
		if got.Name != "Team B" || got.Description != "updated" {
			t.Fatalf("unexpected updated team: %#v", got)
		}
	})
}

func TestBoltStoreDeleteTeamCascadesRelatedRecords(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		team := model.NewTeam("Team A", "", "owner-1")
		if err := s.CreateTeam(team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}

		member := model.NewTeamMember(team.ID, "owner-1", "owner")
		msg := model.NewMessage(team.ID, "owner-1", "Owner", "hello", "text")
		upload := &model.UploadMeta{
			ID:          "upload-1",
			Filename:    "f.png",
			ContentType: "image/png",
			OwnerID:     "owner-1",
			TeamID:      team.ID,
			CreatedAt:   time.Now(),
		}
		putStoreRecord(t, s.db, "team_members", []byte(member.ID), member)
		putStoreRecord(t, s.db, "messages", []byte(msg.ID), msg)
		putStoreRecord(t, s.db, "uploads", []byte(upload.ID), upload)
		err := s.db.Update(func(tx *bbolt.Tx) error {
			return tx.Bucket([]byte("upload_data")).Put([]byte(upload.ID), []byte("data"))
		})
		if err != nil {
			t.Fatalf("put upload data: %v", err)
		}

		if err := s.DeleteTeam(team.ID); err != nil {
			t.Fatalf("DeleteTeam: %v", err)
		}

		if got := mustBucketGet(t, s.db, "teams", []byte(team.ID)); got != nil {
			t.Fatalf("expected team deleted, got %s", string(got))
		}
		if got := mustBucketGet(t, s.db, "uploads", []byte(upload.ID)); got != nil {
			t.Fatalf("expected upload meta deleted")
		}
		if got := mustBucketGet(t, s.db, "upload_data", []byte(upload.ID)); got != nil {
			t.Fatalf("expected upload data deleted")
		}

		// membership + message are keyed by generated IDs, so verify by scanning bucket contents.
		err = s.db.View(func(tx *bbolt.Tx) error {
			for _, bucketName := range []string{"team_members", "messages"} {
				b := tx.Bucket([]byte(bucketName))
				c := b.Cursor()
				for k, v := c.First(); k != nil; k, v = c.Next() {
					switch bucketName {
					case "team_members":
						var m model.TeamMember
						if json.Unmarshal(v, &m) == nil && m.TeamID == team.ID {
							t.Fatalf("expected membership for team %s deleted", team.ID)
						}
					case "messages":
						var m model.Message
						if json.Unmarshal(v, &m) == nil && m.TeamID == team.ID {
							t.Fatalf("expected message for team %s deleted", team.ID)
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("verify cascades: %v", err)
		}
	})
}

func TestBoltStoreTeamMembershipOperations(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		owner := model.NewUser("owner", "Owner", "hashed")
		memberUser := model.NewUser("member", "Member", "hashed")
		if err := s.CreateUser(owner); err != nil {
			t.Fatalf("CreateUser owner: %v", err)
		}
		if err := s.CreateUser(memberUser); err != nil {
			t.Fatalf("CreateUser member: %v", err)
		}

		team := model.NewTeam("Team A", "", owner.ID)
		if err := s.CreateTeam(team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}

		ownerMember := model.NewTeamMember(team.ID, owner.ID, "owner")
		normalMember := model.NewTeamMember(team.ID, memberUser.ID, "member")
		if err := s.AddTeamMember(ownerMember); err != nil {
			t.Fatalf("AddTeamMember owner: %v", err)
		}
		if err := s.AddTeamMember(normalMember); err != nil {
			t.Fatalf("AddTeamMember member: %v", err)
		}

		isMember, err := s.IsTeamMember(team.ID, memberUser.ID)
		if err != nil {
			t.Fatalf("IsTeamMember: %v", err)
		}
		if !isMember {
			t.Fatal("expected member to be in team")
		}

		members, err := s.GetTeamMembers(team.ID)
		if err != nil {
			t.Fatalf("GetTeamMembers: %v", err)
		}
		if len(members) != 2 {
			t.Fatalf("expected 2 members, got %d", len(members))
		}

		gotUser, err := s.GetUserByID(memberUser.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if gotUser.Username != memberUser.Username {
			t.Fatalf("unexpected user: %#v", gotUser)
		}

		if err := s.RemoveTeamMember(team.ID, memberUser.ID); err != nil {
			t.Fatalf("RemoveTeamMember: %v", err)
		}

		isMember, err = s.IsTeamMember(team.ID, memberUser.ID)
		if err != nil {
			t.Fatalf("IsTeamMember after remove: %v", err)
		}
		if isMember {
			t.Fatal("expected member removed from team")
		}
	})
}

func TestBoltStoreTransferTeamOwnership_UpdatesOwnerAndRoles(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		owner := model.NewUser("owner", "Owner", "hashed")
		memberUser := model.NewUser("member", "Member", "hashed")
		_ = s.CreateUser(owner)
		_ = s.CreateUser(memberUser)
		team := model.NewTeam("Team A", "", owner.ID)
		_ = s.CreateTeam(team)
		ownerMember := model.NewTeamMember(team.ID, owner.ID, "owner")
		member := model.NewTeamMember(team.ID, memberUser.ID, "member")
		_ = s.AddTeamMember(ownerMember)
		_ = s.AddTeamMember(member)

		if err := s.TransferTeamOwnership(team.ID, owner.ID, memberUser.ID); err != nil {
			t.Fatalf("TransferTeamOwnership: %v", err)
		}

		updatedTeam, err := s.GetTeamByID(team.ID)
		if err != nil {
			t.Fatalf("GetTeamByID: %v", err)
		}
		if updatedTeam.OwnerID != memberUser.ID {
			t.Fatalf("expected owner %s, got %s", memberUser.ID, updatedTeam.OwnerID)
		}

		members, err := s.GetTeamMembers(team.ID)
		if err != nil {
			t.Fatalf("GetTeamMembers: %v", err)
		}
		roleByUser := map[string]string{}
		for _, m := range members {
			roleByUser[m.UserID] = m.Role
		}
		if roleByUser[memberUser.ID] != "owner" {
			t.Fatalf("expected new owner role 'owner', got %q", roleByUser[memberUser.ID])
		}
		if roleByUser[owner.ID] != "member" {
			t.Fatalf("expected old owner role 'member', got %q", roleByUser[owner.ID])
		}
	})
}

func TestBoltStoreTransferTeamOwnership_RejectsNonMemberTarget(t *testing.T) {
	withTempBoltStore(t, func(s *BoltStore) {
		owner := model.NewUser("owner", "Owner", "hashed")
		outsider := model.NewUser("outsider", "Outsider", "hashed")
		_ = s.CreateUser(owner)
		_ = s.CreateUser(outsider)
		team := model.NewTeam("Team A", "", owner.ID)
		_ = s.CreateTeam(team)
		_ = s.AddTeamMember(model.NewTeamMember(team.ID, owner.ID, "owner"))

		if err := s.TransferTeamOwnership(team.ID, owner.ID, outsider.ID); err == nil {
			t.Fatal("expected transfer to non-member to fail")
		}
	})
}
