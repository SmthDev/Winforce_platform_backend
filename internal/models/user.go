package models

type Profile struct {
	ID		string `json:"id"`
	Email		string `json:"email,omitempty"`
	FirstName	string `json:"first_name,omitempty"`
	LastName	string `json:"last_name,omitempty"`
	AvatarLink	string `json:"avatar_link,omitempty"`
 }