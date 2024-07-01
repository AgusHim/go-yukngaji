package presence

import (
	"errors"
	"mainyuk/internal/event"
	"mainyuk/internal/user"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	PresenceRepository Repository
	UserService        user.Service
	EventService       event.Service
}

func NewService(repository Repository, userService user.Service, eventService event.Service) Service {
	return &service{
		PresenceRepository: repository,
		UserService:        userService,
		EventService:       eventService,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreatePresence) (*Presence, error) {
	if req.User == nil && req.UserID == nil {
		return nil, errors.New("InvalidRequest")
	}

	event, errEvent := s.EventService.Show(c, req.EventID)
	if errEvent != nil {
		return nil, errors.New("EventNotFound")
	}

	if req.UserID != nil {
		presence, _ := s.PresenceRepository.FindByUserID(c, *req.UserID, event.ID)
		if presence != nil {
			return presence, nil
		}
	}

	presence := &Presence{}
	presence.Event = event
	presence.ID = uuid.NewString()
	presence.EventID = event.ID

	var user *user.User
	if req.UserID == nil && req.User != nil {
		u, err := s.UserService.Presence(c, req.User)
		if err != nil {
			return nil, err
		}
		presence.UserID = u.ID
		user = u
	}
	if req.UserID != nil && req.User == nil {
		u, errUser := s.UserService.Show(c, *req.UserID)
		if errUser != nil {
			return nil, errors.New("UserNotFound")
		}
		presence.UserID = u.ID
		user = u
	}

	presence.User = user
	presence.CreatedAt = time.Now()
	presence.UpdatedAt = time.Now()

	presence, err := s.PresenceRepository.Create(c, presence)
	if err != nil {
		// Delete if user created
		if user != nil {
			if err := s.UserService.DeleteByID(c, user.ID); err != nil {
				return nil, err
			}
		}
		return nil, err
	}
	// ++ participant from event
	event.Participant = event.Participant + 1
	_, errUpdateEvent := s.EventService.Update(c, event.ID, event)
	if errUpdateEvent != nil {
		return nil, err
	}

	return presence, nil
}

func (s *service) Show(c *gin.Context, id string) (*Presence, error) {
	presence, err := s.PresenceRepository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (s *service) Index(c *gin.Context) ([]*Presence, error) {
	presence, err := s.PresenceRepository.Index(c)
	if err != nil {
		return nil, err
	}
	return presence, nil
}
