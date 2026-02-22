package main

import (
	"encoding/json"
	"testing"
)

func TestBroadcastUserProfileUpdated_UpdatesConnectedClientAndBroadcasts(t *testing.T) {
	withTempDB(t, func() {
		origHub := hub
		hub = &Hub{Rooms: make(map[string]*TeamRoom)}
		defer func() { hub = origHub }()

		user := NewUser("alice", "Alice", "hashed")
		observer := NewUser("bob", "Bob", "hashed")
		if err := createUser(user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		if err := createUser(observer); err != nil {
			t.Fatalf("create observer: %v", err)
		}
		team := NewTeam("Team", "", user.ID)
		if err := createTeam(team); err != nil {
			t.Fatalf("create team: %v", err)
		}
		_ = addTeamMember(NewTeamMember(team.ID, user.ID, "owner"))
		_ = addTeamMember(NewTeamMember(team.ID, observer.ID, "member"))

		targetClient := &Client{UserID: user.ID, Username: "Alice", TeamID: team.ID, Send: make(chan []byte, 2)}
		observerClient := &Client{UserID: observer.ID, Username: "Bob", TeamID: team.ID, Send: make(chan []byte, 2)}
		room := &TeamRoom{ID: team.ID, Clients: map[*Client]bool{targetClient: true, observerClient: true}}
		hub.Rooms[team.ID] = room

		broadcastUserProfileUpdated(user.ID, user.Username, "Alice Renamed")

		if targetClient.Username != "Alice Renamed" {
			t.Fatalf("expected client display name updated, got %q", targetClient.Username)
		}

		var msg WebSocketMessage
		select {
		case raw := <-observerClient.Send:
			if err := json.Unmarshal(raw, &msg); err != nil {
				t.Fatalf("unmarshal ws message: %v", err)
			}
		default:
			t.Fatal("expected profile update websocket broadcast")
		}

		if msg.Type != "user_profile_updated" {
			t.Fatalf("expected user_profile_updated event, got %q", msg.Type)
		}
	})
}

