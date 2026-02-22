package main

import "SuperChat/internal/model"

type User = model.User
type Team = model.Team
type TeamMember = model.TeamMember
type Message = model.Message
type LoginRequest = model.LoginRequest
type RegisterRequest = model.RegisterRequest
type CreateTeamRequest = model.CreateTeamRequest
type SendMessageRequest = model.SendMessageRequest
type APIResponse = model.APIResponse
type WebSocketMessage = model.WebSocketMessage
type UploadMeta = model.UploadMeta

func NewUser(username, name, password string) *User {
	return model.NewUser(username, name, password)
}

func NewTeam(name, description, ownerID string) *Team {
	return model.NewTeam(name, description, ownerID)
}

func NewMessage(teamID, userID, username, content, msgType string) *Message {
	return model.NewMessage(teamID, userID, username, content, msgType)
}

func NewTeamMember(teamID, userID, role string) *TeamMember {
	return model.NewTeamMember(teamID, userID, role)
}
