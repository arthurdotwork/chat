package domain

import "github.com/google/uuid"

type Message struct {
	ID      uuid.UUID
	Content string
	Sender  User
}
