package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"primaryKey"`
	Password string
}

func (u User) String() string {
	return fmt.Sprintf("User{Username: %s, Password: N/A}", u.Username)
}

type Post struct {
	gorm.Model
	Username  string
	Message   string
	Timestamp time.Time
}

func (p Post) String() string {
	return fmt.Sprintf("Post{Username: %s, Message: %s, Timestamp: %d}",
		p.Username, p.Message, p.Timestamp.Unix())
}

type Following struct {
	gorm.Model
	Username string
	Follower string
}

func (f Following) String() string {
	return fmt.Sprintf("Following{Username: %s, Follower: %s}", f.Username, f.Follower)
}

type Auth struct {
	gorm.Model
	Username string `gorm:"primaryKey"`
	AuthId   string
}

func (a Auth) String() string {
	return fmt.Sprintf("Auth{Username: %s, AuthId: %s}", a.Username, a.AuthId)
}

type Paste struct {
	gorm.Model
	Username string `gorm:"primaryKey"`
	Title    string
	Text     string
}

func (p Paste) String() string {
	return fmt.Sprintf("Paste{Username: %s, Title: %s, Text: %q}", p.Username, p.Title, p.Text)
}

func (p *Paste) Validate() error {
	if p.Username == "" {
		return fmt.Errorf("paste username cannot be empty string")
	}
	if p.Title == "" {
		return fmt.Errorf("paste title cannot be empty string")
	}
	if p.Text == "" {
		return fmt.Errorf("paste text cannot be empty string")
	}
	return nil
}
