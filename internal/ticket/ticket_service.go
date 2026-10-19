package ticket

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type service struct {
	Repository
}

func NewService(repository Repository) Service {
	return &service{
		repository,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateTicket) (*Ticket, error) {
	ticket := &Ticket{}
	ticket.ID = uuid.NewString()
	ticket.Name = req.Name
	ticket.Description = req.Description
	ticket.Price = req.Price
	ticket.EventID = req.EventID
	ticket.PaxMultiplier = req.PaxMultiplier
	ticket.MinOrderPax = req.MinOrderPax
	ticket.MaxOrderPax = req.MaxOrderPax
	ticket.MaxPax = req.MaxPax
	ticket.AccessMode = "all"
	if req.AccessMode != nil {
		ticket.AccessMode = strings.ToLower(*req.AccessMode)
	}
	if err := validateAccess(ticket.AccessMode, req.EpisodeIDs); err != nil {
		return nil, err
	}

	startAt, errParsed := time.Parse("2006-01-02T15:04", req.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	endAt, errParsed := time.Parse("2006-01-02T15:04", req.EndAt)
	if errParsed != nil {
		return nil, errParsed
	}
	ticket.StartAt = startAt
	ticket.EndAt = endAt

	ticket.Visibility = "draft"
	if req.Visibility != nil {
		ticket.Visibility = strings.ToLower(*req.Visibility)
	}

	ticket.GenderAllowed = "both"
	if req.GenderAllowed != nil {
		ticket.GenderAllowed = strings.ToLower(*req.GenderAllowed)
	}

	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	ticket, err := s.Repository.Create(c, ticket)
	if err != nil {
		return nil, err
	}
	if err := s.syncEpisodeAccess(ticket.ID, ticket.EventID, ticket.AccessMode, req.EpisodeIDs); err != nil {
		return nil, err
	}
	ticket.EpisodeIDs = req.EpisodeIDs
	return ticket, nil
}

func (s *service) Update(c *gin.Context, id string, req *CreateTicket) (*Ticket, error) {
	ticket, err := s.Show(c, id)
	if err != nil {
		return nil, err
	}

	ticket.Name = req.Name
	ticket.Description = req.Description
	ticket.Price = req.Price
	ticket.EventID = req.EventID
	ticket.PaxMultiplier = req.PaxMultiplier
	ticket.MinOrderPax = req.MinOrderPax
	ticket.MaxOrderPax = req.MaxOrderPax
	ticket.MaxPax = req.MaxPax
	if req.AccessMode != nil {
		ticket.AccessMode = strings.ToLower(*req.AccessMode)
	}
	if err := validateAccess(ticket.AccessMode, req.EpisodeIDs); err != nil {
		return nil, err
	}

	startAt, errParsed := time.Parse("2006-01-02T15:04", req.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	endAt, errParsed := time.Parse("2006-01-02T15:04", req.EndAt)
	if errParsed != nil {
		return nil, errParsed
	}
	ticket.StartAt = startAt
	ticket.EndAt = endAt

	if req.Visibility != nil {
		ticket.Visibility = strings.ToLower(*req.Visibility)
	}
	if req.GenderAllowed != nil {
		ticket.GenderAllowed = strings.ToLower(*req.GenderAllowed)
	}

	ticket.UpdatedAt = time.Now()

	updatedTicket, err := s.Repository.Update(c, ticket.ID, ticket)
	if err != nil {
		return nil, err
	}
	if err := s.syncEpisodeAccess(ticket.ID, ticket.EventID, ticket.AccessMode, req.EpisodeIDs); err != nil {
		return nil, err
	}
	updatedTicket.EpisodeIDs = req.EpisodeIDs
	return updatedTicket, nil
}

func validateAccess(mode string, episodeIDs []string) error {
	if mode != "all" && mode != "selected" {
		return fmt.Errorf("access_mode harus all atau selected")
	}
	if mode == "selected" && len(episodeIDs) == 0 {
		return fmt.Errorf("pilih minimal satu episode untuk tiket selected")
	}
	return nil
}

func uniqueEpisodeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

type ticketEpisodeAccess struct {
	TicketID  string `gorm:"column:ticket_id"`
	EpisodeID string `gorm:"column:episode_id"`
}

func (ticketEpisodeAccess) TableName() string { return "ticket_episode_access" }

func (s *service) syncEpisodeAccess(ticketID, eventID, mode string, episodeIDs []string) error {
	episodeIDs = uniqueEpisodeIDs(episodeIDs)
	db := s.repositoryDB()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&ticketEpisodeAccess{}).Error; err != nil {
			return err
		}
		if mode == "all" {
			return nil
		}
		var count int64
		if err := tx.Model(&Episode{}).Where("event_id = ? AND id IN ? AND deleted_at IS NULL", eventID, episodeIDs).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(episodeIDs)) {
			return fmt.Errorf("episode harus milik event yang sama")
		}
		for _, episodeID := range episodeIDs {
			if err := tx.Create(&ticketEpisodeAccess{TicketID: ticketID, EpisodeID: episodeID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *service) Show(c *gin.Context, id string) (*Ticket, error) {
	divisi, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Index(c *gin.Context) ([]*Ticket, error) {
	divisi, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Delete(c *gin.Context, id string) error {
	ticket, err := s.Show(c, id)
	if err != nil {
		return err
	}

	now := time.Now()
	ticket.UpdatedAt = now
	ticket.DeletedAt = &now

	_, err = s.Repository.Update(c, ticket.ID, ticket)
	if err != nil {
		return err
	}
	return nil
}
