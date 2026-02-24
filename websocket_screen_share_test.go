package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestScreenShareRegistryReserveSessionOnEmptyTeam(t *testing.T) {
	r := newScreenShareRegistry()

	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session")
	}
	if sess.TeamID != "team-1" || sess.SharerUserID != "user-1" || sess.SharerName != "Alice" {
		t.Fatalf("unexpected session: %#v", sess)
	}
	if sess.SessionID == "" {
		t.Fatal("expected non-empty session id")
	}
	if sess.Status != screenShareStatusReserved {
		t.Fatalf("expected reserved status, got %q", sess.Status)
	}
}

func TestScreenShareRegistryDenySecondSharerAndReturnCurrentSharer(t *testing.T) {
	r := newScreenShareRegistry()

	if _, err := r.reserve("team-1", "user-1", "Alice"); err != nil {
		t.Fatalf("first reserve: %v", err)
	}

	_, err := r.reserve("team-1", "user-2", "Bob")
	if err == nil {
		t.Fatal("expected second reserve to fail")
	}

	alreadyActive, ok := err.(*screenShareAlreadyActiveError)
	if !ok {
		t.Fatalf("expected *screenShareAlreadyActiveError, got %T (%v)", err, err)
	}
	if alreadyActive.SharerUserID != "user-1" || alreadyActive.SharerName != "Alice" {
		t.Fatalf("unexpected active sharer info: %#v", alreadyActive)
	}
}

func TestScreenShareRegistryReleaseClearsLock(t *testing.T) {
	r := newScreenShareRegistry()

	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	if ok := r.release("team-1", sess.SessionID); !ok {
		t.Fatal("expected release to succeed")
	}

	if got := r.get("team-1"); got != nil {
		t.Fatalf("expected no session after release, got %#v", got)
	}

	if _, err := r.reserve("team-1", "user-2", "Bob"); err != nil {
		t.Fatalf("expected reserve after release to succeed, got %v", err)
	}
}

func TestScreenShareRegistryValidatePublisherAttachRejectsWrongUser(t *testing.T) {
	r := newScreenShareRegistry()

	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	if err := r.validatePublisherAttach("team-1", sess.SessionID, "user-2"); err == nil {
		t.Fatal("expected wrong user attach to fail")
	}
}

func TestScreenShareControlRequestStartGrantsWhenNoActiveSharer(t *testing.T) {
	orig := screenShareSessions
	screenShareSessions = newScreenShareRegistry()
	defer func() { screenShareSessions = orig }()

	room := &TeamRoom{ID: "team-1", Clients: map[*Client]bool{}}
	client := &Client{UserID: "user-1", Username: "Alice", TeamID: "team-1", Send: make(chan []byte, 4)}
	room.Clients[client] = true

	called := client.handleScreenShareRequestStart(room)
	if !called {
		t.Fatal("expected request start handler to handle message")
	}

	msg := mustReadWSMessage(t, client.Send)
	if msg.Type != "screen_share_start_granted" {
		t.Fatalf("expected grant event, got %q", msg.Type)
	}
	payload := payloadMap(t, msg)
	if payload["session_id"] == "" {
		t.Fatalf("expected session_id in payload, got %#v", payload)
	}
	if payload["sharer_name"] != "Alice" {
		t.Fatalf("expected sharer_name Alice, got %#v", payload["sharer_name"])
	}
}

func TestScreenShareControlRequestStartDeniesWhenActiveSharerExists(t *testing.T) {
	orig := screenShareSessions
	screenShareSessions = newScreenShareRegistry()
	defer func() { screenShareSessions = orig }()

	room := &TeamRoom{ID: "team-1", Clients: map[*Client]bool{}}
	alice := &Client{UserID: "user-1", Username: "Alice", TeamID: "team-1", Send: make(chan []byte, 4)}
	bob := &Client{UserID: "user-2", Username: "Bob", TeamID: "team-1", Send: make(chan []byte, 4)}
	room.Clients[alice] = true
	room.Clients[bob] = true

	if !alice.handleScreenShareRequestStart(room) {
		t.Fatal("expected alice start request handled")
	}
	_ = mustReadWSMessage(t, alice.Send) // drain grant

	if !bob.handleScreenShareRequestStart(room) {
		t.Fatal("expected bob start request handled")
	}
	msg := mustReadWSMessage(t, bob.Send)
	if msg.Type != "screen_share_denied" {
		t.Fatalf("expected denied event, got %q", msg.Type)
	}
	payload := payloadMap(t, msg)
	if payload["sharer_name"] != "Alice" || payload["sharer_user_id"] != "user-1" {
		t.Fatalf("unexpected deny payload: %#v", payload)
	}
}

func TestScreenShareControlStopFromSharerReleasesSessionAndBroadcasts(t *testing.T) {
	orig := screenShareSessions
	screenShareSessions = newScreenShareRegistry()
	defer func() { screenShareSessions = orig }()

	room := &TeamRoom{ID: "team-1", Clients: map[*Client]bool{}}
	alice := &Client{UserID: "user-1", Username: "Alice", TeamID: "team-1", Send: make(chan []byte, 8)}
	bob := &Client{UserID: "user-2", Username: "Bob", TeamID: "team-1", Send: make(chan []byte, 8)}
	room.Clients[alice] = true
	room.Clients[bob] = true

	if !alice.handleScreenShareRequestStart(room) {
		t.Fatal("expected start request handled")
	}
	grant := mustReadWSMessage(t, alice.Send)
	sessionID := payloadMap(t, grant)["session_id"]
	if sessionID == "" {
		t.Fatal("expected session id")
	}

	if !alice.handleScreenShareStop(room) {
		t.Fatal("expected stop request handled")
	}
	stopMsg := mustReadWSMessage(t, bob.Send)
	if stopMsg.Type != "screen_share_stopped" {
		t.Fatalf("expected stop broadcast, got %q", stopMsg.Type)
	}
	if got := screenShareSessions.get("team-1"); got != nil {
		t.Fatalf("expected session released, got %#v", got)
	}
}

func TestScreenShareControlStopFromNonSharerDoesNotReleaseSession(t *testing.T) {
	orig := screenShareSessions
	screenShareSessions = newScreenShareRegistry()
	defer func() { screenShareSessions = orig }()

	room := &TeamRoom{ID: "team-1", Clients: map[*Client]bool{}}
	alice := &Client{UserID: "user-1", Username: "Alice", TeamID: "team-1", Send: make(chan []byte, 8)}
	bob := &Client{UserID: "user-2", Username: "Bob", TeamID: "team-1", Send: make(chan []byte, 8)}
	room.Clients[alice] = true
	room.Clients[bob] = true

	if !alice.handleScreenShareRequestStart(room) {
		t.Fatal("expected start request handled")
	}
	_ = mustReadWSMessage(t, alice.Send)

	if !bob.handleScreenShareStop(room) {
		t.Fatal("expected stop handler to handle and reject safely")
	}
	if got := screenShareSessions.get("team-1"); got == nil {
		t.Fatal("expected session to remain active after non-sharer stop")
	}
	msg := mustReadWSMessage(t, bob.Send)
	if msg.Type != "screen_share_error" {
		t.Fatalf("expected error response, got %q", msg.Type)
	}
}

func mustReadWSMessage(t *testing.T, ch <-chan []byte) WebSocketMessage {
	t.Helper()
	select {
	case raw := <-ch:
		var msg WebSocketMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal websocket message: %v", err)
		}
		return msg
	default:
		t.Fatal("expected websocket message")
		return WebSocketMessage{}
	}
}

func payloadMap(t *testing.T, msg WebSocketMessage) map[string]interface{} {
	t.Helper()
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected payload map, got %T", msg.Payload)
	}
	return payload
}

func TestScreenShareEndpointRejectsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/ws/team-1/screen?role=viewer&session_id=s1", nil)
	req = mux.SetURLVars(req, map[string]string{"teamId": "team-1"})
	rr := httptest.NewRecorder()

	handleScreenStreamWebSocket(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestScreenShareEndpointRejectsNonMember(t *testing.T) {
	withTempDB(t, func() {
		user := NewUser("alice", "Alice", "hashed")
		if err := createUser(user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		team := NewTeam("Team", "", "other-owner")
		if err := createTeam(team); err != nil {
			t.Fatalf("create team: %v", err)
		}

		token, err := generateToken(user)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/ws/"+team.ID+"/screen?token="+token+"&role=viewer&session_id=s1", nil)
		req = mux.SetURLVars(req, map[string]string{"teamId": team.ID})
		rr := httptest.NewRecorder()

		handleScreenStreamWebSocket(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (%s)", rr.Code, rr.Body.String())
		}
	})
}

func TestScreenShareEndpointRejectsViewerWhenNoActiveSession(t *testing.T) {
	withTempDB(t, func() {
		user, team, token := setupScreenShareEndpointMember(t)

		req := httptest.NewRequest(http.MethodGet, "/api/ws/"+team.ID+"/screen?token="+token+"&role=viewer&session_id=missing", nil)
		req = mux.SetURLVars(req, map[string]string{"teamId": team.ID})
		rr := httptest.NewRecorder()

		handleScreenStreamWebSocket(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d (%s)", rr.Code, rr.Body.String())
		}
		_ = user
	})
}

func TestScreenShareEndpointRejectsPublisherSessionUserMismatch(t *testing.T) {
	withTempDB(t, func() {
		orig := screenShareSessions
		screenShareSessions = newScreenShareRegistry()
		defer func() { screenShareSessions = orig }()

		alice := NewUser("alice", "Alice", "hashed")
		bob := NewUser("bob", "Bob", "hashed")
		for _, u := range []*User{alice, bob} {
			if err := createUser(u); err != nil {
				t.Fatalf("create user %s: %v", u.Username, err)
			}
		}
		team := NewTeam("Team", "", alice.ID)
		if err := createTeam(team); err != nil {
			t.Fatalf("create team: %v", err)
		}
		_ = addTeamMember(NewTeamMember(team.ID, alice.ID, "owner"))
		_ = addTeamMember(NewTeamMember(team.ID, bob.ID, "member"))

		sess, err := screenShareSessions.reserve(team.ID, alice.ID, "Alice")
		if err != nil {
			t.Fatalf("reserve: %v", err)
		}

		bobToken, err := generateToken(bob)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/ws/"+team.ID+"/screen?token="+bobToken+"&role=publisher&session_id="+sess.SessionID,
			nil,
		)
		req = mux.SetURLVars(req, map[string]string{"teamId": team.ID})
		rr := httptest.NewRecorder()

		handleScreenStreamWebSocket(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (%s)", rr.Code, rr.Body.String())
		}
	})
}

func setupScreenShareEndpointMember(t *testing.T) (*User, *Team, string) {
	t.Helper()

	user := NewUser("alice", "Alice", "hashed")
	if err := createUser(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	team := NewTeam("Team", "", user.ID)
	if err := createTeam(team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	_ = addTeamMember(NewTeamMember(team.ID, user.ID, "owner"))
	token, err := generateToken(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return user, team, token
}

func TestScreenShareRelayPublisherInitStoresMetadataAndMarksActive(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	initMsg, err := r.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType:  "video/webm;codecs=vp8",
		Width:     1280,
		Height:    720,
		FPSTarget: 30,
	})
	if err != nil {
		t.Fatalf("publisherInit: %v", err)
	}
	if len(initMsg) == 0 {
		t.Fatal("expected serialized init payload")
	}

	got := r.get("team-1")
	if got == nil {
		t.Fatal("expected session")
	}
	if got.Status != screenShareStatusActive {
		t.Fatalf("expected active status, got %q", got.Status)
	}
	if got.MimeType != "video/webm;codecs=vp8" || got.Width != 1280 || got.Height != 720 || got.FPSTarget != 30 {
		t.Fatalf("unexpected metadata on session: %#v", got)
	}
}

func TestScreenShareRelayViewerReceivesInitWhenJoiningActiveSession(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := r.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType:  "video/webm;codecs=vp8",
		Width:     1280,
		Height:    720,
		FPSTarget: 30,
	}); err != nil {
		t.Fatalf("publisherInit: %v", err)
	}

	viewerCh := make(chan screenShareOutboundMessage, 2)
	if err := r.addViewer("team-1", sess.SessionID, "viewer-1", viewerCh); err != nil {
		t.Fatalf("addViewer: %v", err)
	}

	msg := mustReadScreenShareOutbound(t, viewerCh)
	if msg.Kind != screenShareOutboundInit {
		t.Fatalf("expected init message, got %q", msg.Kind)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		t.Fatalf("unmarshal init payload: %v", err)
	}
	if payload["type"] != "init" || payload["mime_type"] != "video/webm;codecs=vp8" {
		t.Fatalf("unexpected init payload: %#v", payload)
	}
}

func TestScreenShareRelayBinaryChunksRelayToAllViewers(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := r.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType:  "video/webm",
		Width:     800,
		Height:    600,
		FPSTarget: 30,
	}); err != nil {
		t.Fatalf("publisherInit: %v", err)
	}

	ch1 := make(chan screenShareOutboundMessage, 4)
	ch2 := make(chan screenShareOutboundMessage, 4)
	if err := r.addViewer("team-1", sess.SessionID, "v1", ch1); err != nil {
		t.Fatalf("addViewer v1: %v", err)
	}
	if err := r.addViewer("team-1", sess.SessionID, "v2", ch2); err != nil {
		t.Fatalf("addViewer v2: %v", err)
	}
	_ = mustReadScreenShareOutbound(t, ch1) // initial init
	_ = mustReadScreenShareOutbound(t, ch2)

	chunk := []byte{0x01, 0x02, 0x03, 0x04}
	if err := r.relayPublisherChunk("team-1", sess.SessionID, "user-1", chunk); err != nil {
		t.Fatalf("relayPublisherChunk: %v", err)
	}

	got1 := mustReadScreenShareOutbound(t, ch1)
	got2 := mustReadScreenShareOutbound(t, ch2)
	if got1.Kind != screenShareOutboundChunk || got2.Kind != screenShareOutboundChunk {
		t.Fatalf("expected chunk messages, got %q and %q", got1.Kind, got2.Kind)
	}
	if string(got1.Data) != string(chunk) || string(got2.Data) != string(chunk) {
		t.Fatalf("unexpected chunk payloads: %v %v", got1.Data, got2.Data)
	}
}

func TestScreenShareRelayOversizedChunkRejectedAndDropped(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := r.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType:  "video/webm",
		Width:     800,
		Height:    600,
		FPSTarget: 30,
	}); err != nil {
		t.Fatalf("publisherInit: %v", err)
	}

	viewerCh := make(chan screenShareOutboundMessage, 4)
	if err := r.addViewer("team-1", sess.SessionID, "v1", viewerCh); err != nil {
		t.Fatalf("addViewer: %v", err)
	}
	_ = mustReadScreenShareOutbound(t, viewerCh) // init

	oversized := make([]byte, screenShareMaxChunkBytes+1)
	err = r.relayPublisherChunk("team-1", sess.SessionID, "user-1", oversized)
	if err == nil {
		t.Fatal("expected oversized chunk error")
	}
	if !errorsIsScreenShareChunkTooLarge(err) {
		t.Fatalf("expected chunk too large error, got %v", err)
	}
	select {
	case msg := <-viewerCh:
		t.Fatalf("expected no chunk relayed, got %#v", msg)
	default:
	}
}

func TestScreenShareRelayChunkBeforeInitDoesNotCrash(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	if err := r.relayPublisherChunk("team-1", sess.SessionID, "user-1", []byte{1, 2, 3}); err == nil {
		t.Fatal("expected error when relaying chunk before init")
	}
}

func mustReadScreenShareOutbound(t *testing.T, ch <-chan screenShareOutboundMessage) screenShareOutboundMessage {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	default:
		t.Fatal("expected screen share outbound message")
		return screenShareOutboundMessage{}
	}
}

func errorsIsScreenShareChunkTooLarge(err error) bool {
	return errors.Is(err, errScreenShareChunkTooLarge)
}

func TestScreenShareCleanupReservedSessionExpiresIfPublisherNeverConnects(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	expired := r.expireStale(time.Now().Add(11*time.Second), 10*time.Second, 5*time.Second)
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired session, got %d", len(expired))
	}
	if expired[0].SessionID != sess.SessionID {
		t.Fatalf("unexpected expired session: %#v", expired[0])
	}
	if got := r.get("team-1"); got != nil {
		t.Fatalf("expected session removed after expiry, got %#v", got)
	}
}

func TestScreenShareCleanupActiveSessionExpiresOnIdleTimeout(t *testing.T) {
	r := newScreenShareRegistry()
	sess, err := r.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := r.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType:  "video/webm",
		Width:     1280,
		Height:    720,
		FPSTarget: 30,
	}); err != nil {
		t.Fatalf("publisherInit: %v", err)
	}

	expired := r.expireStale(time.Now().Add(6*time.Second), 10*time.Second, 5*time.Second)
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired session, got %d", len(expired))
	}
	if got := r.get("team-1"); got != nil {
		t.Fatalf("expected active session removed after idle expiry, got %#v", got)
	}
}

func TestScreenShareCleanupPublisherDisconnectReleasesSessionAndBroadcastsStop(t *testing.T) {
	orig := screenShareSessions
	screenShareSessions = newScreenShareRegistry()
	defer func() { screenShareSessions = orig }()

	room := &TeamRoom{ID: "team-1", Clients: map[*Client]bool{}}
	watcher := &Client{UserID: "user-2", Username: "Bob", TeamID: "team-1", Send: make(chan []byte, 4)}
	room.Clients[watcher] = true

	sess, err := screenShareSessions.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := screenShareSessions.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType: "video/webm",
	}); err != nil {
		t.Fatalf("publisherInit: %v", err)
	}

	cleaned := handleScreenSharePublisherDisconnected(room, "team-1", sess.SessionID, "user-1")
	if !cleaned {
		t.Fatal("expected publisher disconnect cleanup")
	}
	if got := screenShareSessions.get("team-1"); got != nil {
		t.Fatalf("expected session released, got %#v", got)
	}
	msg := mustReadWSMessage(t, watcher.Send)
	if msg.Type != "screen_share_stopped" {
		t.Fatalf("expected stop broadcast, got %q", msg.Type)
	}
}

func TestScreenShareCleanupSharerChatDisconnectReleasesSessionAndBroadcastsStop(t *testing.T) {
	orig := screenShareSessions
	screenShareSessions = newScreenShareRegistry()
	defer func() { screenShareSessions = orig }()

	room := &TeamRoom{ID: "team-1", Clients: map[*Client]bool{}}
	sharer := &Client{UserID: "user-1", Username: "Alice", TeamID: "team-1", Send: make(chan []byte, 4)}
	watcher := &Client{UserID: "user-2", Username: "Bob", TeamID: "team-1", Send: make(chan []byte, 4)}
	room.Clients[sharer] = true
	room.Clients[watcher] = true

	sess, err := screenShareSessions.reserve("team-1", "user-1", "Alice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := screenShareSessions.publisherInit("team-1", sess.SessionID, "user-1", screenShareInit{
		MimeType: "video/webm",
	}); err != nil {
		t.Fatalf("publisherInit: %v", err)
	}

	// Simulate the disconnect helper being called after sharer is removed from room.
	delete(room.Clients, sharer)
	cleaned := releaseScreenShareForChatClientDisconnect(room, sharer)
	if !cleaned {
		t.Fatal("expected sharer disconnect cleanup")
	}
	if got := screenShareSessions.get("team-1"); got != nil {
		t.Fatalf("expected session released, got %#v", got)
	}
	msg := mustReadWSMessage(t, watcher.Send)
	if msg.Type != "screen_share_stopped" {
		t.Fatalf("expected stop broadcast, got %q", msg.Type)
	}
}
