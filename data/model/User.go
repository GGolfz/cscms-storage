package model

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey,autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Provider  string    `json:"provider"`
	AvatarURL string    `json:"avatar_url"`
	Files     []File    `json:"files"`
	Images    []Image   `json:"images"`
	APIKey    string    `json:"api_key" gorm:"index"`
}
