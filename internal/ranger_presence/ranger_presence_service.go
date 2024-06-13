package ranger_presence

import (
	"errors"
	"mainyuk/internal/agenda"
	"mainyuk/internal/divisi"
	"mainyuk/internal/ranger"
	"mainyuk/internal/user"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository    Repository
	RangerService ranger.Service
	AgendaService agenda.Service
	DivisiService divisi.Service
}

func NewService(repository Repository, rangerService ranger.Service, agendaService agenda.Service, divisiService divisi.Service) Service {
	return &service{
		Repository:    repository,
		RangerService: rangerService,
		AgendaService: agendaService,
		DivisiService: divisiService,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreatePresence) (*RangerPresence, error) {
	presence := &RangerPresence{}
	presence.ID = uuid.NewString()

	p, _ := s.Repository.CheckAlreadyPresence(c, req.RangerID, req.AgendaID)
	if p != nil {
		return p, nil
	}
	ranger, errRanger := s.RangerService.Show(c, req.RangerID)
	if errRanger != nil {
		return nil, errRanger
	}
	presence.Ranger = ranger
	presence.RangerID = ranger.ID

	agenda, errAgenda := s.AgendaService.Show(c, req.AgendaID)
	if errAgenda != nil {
		return nil, errAgenda
	}
	presence.Agenda = agenda
	presence.AgendaID = agenda.ID

	divisi, errDivisi := s.DivisiService.Show(c, req.DivisiID)
	if errDivisi != nil {
		return nil, errDivisi
	}
	presence.Divisi = divisi
	presence.DivisiID = divisi.ID

	presence.CreatedAt = time.Now()
	presence.UpdatedAt = time.Now()

	presence, err := s.Repository.Create(c, presence)
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (s *service) Show(c *gin.Context, id string) (*RangerPresence, error) {
	presence, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (s *service) CheckAlreadyPresence(c *gin.Context, rangerID string, agendaID string) (*RangerPresence, error) {
	presence, err := s.Repository.CheckAlreadyPresence(c, rangerID, agendaID)
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (s *service) Index(c *gin.Context) ([]*RangerPresence, error) {
	if strings.Contains(c.FullPath(), "ranger_api") {
		u, exists := c.Get("currentUser")
		if !exists {

			return nil, errors.New("NotAuthorized")
		}

		currentUser, ok := u.(user.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "FailedParsing: current user",
			})
			return nil, errors.New("ErrorParseCurrentUser")
		}

		ranger, errRanger := s.RangerService.ShowByUserID(c, currentUser.ID)
		if errRanger != nil {
			return nil, errRanger
		}

		presence, err := s.Repository.IndexByUserID(c, ranger.ID)
		if err != nil {
			return nil, err
		}
		return presence, nil
	}
	presence, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return presence, nil
}
