package store

import (
	"context"
	"sync"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/google/uuid"
)

type MemoryRoomStore struct {
	connectedUsers map[uuid.UUID]domain.User
	sync.RWMutex
}

func NewMemoryRoomStore() *MemoryRoomStore {
	return &MemoryRoomStore{
		connectedUsers: make(map[uuid.UUID]domain.User),
	}
}

func (s *MemoryRoomStore) GetUser(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	s.RLock()
	defer s.RUnlock()

	for _, user := range s.connectedUsers {
		if user.ID == userID {
			return user, nil
		}
	}

	return domain.User{}, nil
}

func (s *MemoryRoomStore) Connect(ctx context.Context, user domain.User) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.connectedUsers[user.ID]; ok {
		return nil
	}

	s.connectedUsers[user.ID] = user
	return nil
}

func (s *MemoryRoomStore) Disconnect(ctx context.Context, user domain.User) error {
	s.Lock()
	defer s.Unlock()

	for i, u := range s.connectedUsers {
		if u.ID == user.ID {
			delete(s.connectedUsers, i)
			break
		}
	}

	return nil
}

func (s *MemoryRoomStore) GetConnectedUsers(ctx context.Context) ([]domain.User, error) {
	s.RLock()
	defer s.RUnlock()

	connectedUsers := make([]domain.User, 0, len(s.connectedUsers))
	for _, user := range s.connectedUsers {
		connectedUsers = append(connectedUsers, user)
	}

	return connectedUsers, nil
}
