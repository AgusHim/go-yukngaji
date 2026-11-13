package event_verifier

import (
	"time"

	"mainyuk/internal/presence"

	"github.com/gin-gonic/gin"
)

const sessionLifetime = 12 * time.Hour

type Credential struct {
	EventID      string    `gorm:"column:event_id;primaryKey"`
	PasswordHash string    `gorm:"column:password_hash"`
	Enabled      bool      `gorm:"column:enabled"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (Credential) TableName() string { return "event_verifier_credentials" }

type Session struct {
	ID         string     `gorm:"column:id;primaryKey"`
	EventID    string     `gorm:"column:event_id"`
	TokenHash  string     `gorm:"column:token_hash"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	LastUsedAt time.Time  `gorm:"column:last_used_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (Session) TableName() string { return "event_verifier_sessions" }

type TicketRow struct {
	PublicID      string     `json:"public_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Phone         string     `json:"phone"`
	Gender        string     `json:"gender"`
	TicketName    string     `json:"ticket_name"`
	CheckedIn     bool       `json:"checked_in"`
	LastCheckInAt *time.Time `json:"last_check_in_at"`
}

type TicketOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Distribution merangkum peserta bertiket paid untuk kebutuhan operasional
// verifikator. Data ini tidak memuat identitas peserta satu per satu.
type Distribution struct {
	TotalTickets int64                 `json:"total_tickets"`
	CheckedIn    int64                 `json:"checked_in"`
	NotCheckedIn int64                 `json:"not_checked_in"`
	Male         int64                 `json:"male"`
	Female       int64                 `json:"female"`
	Unspecified  int64                 `json:"unspecified"`
	ByTicket     []*TicketDistribution `json:"by_ticket"`
}

type TicketDistribution struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TotalTickets int64  `json:"total_tickets"`
	CheckedIn    int64  `json:"checked_in"`
	NotCheckedIn int64  `json:"not_checked_in"`
	Male         int64  `json:"male"`
	Female       int64  `json:"female"`
	Unspecified  int64  `json:"unspecified"`
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	EventID   string    `json:"event_id"`
	EventSlug string    `json:"event_slug"`
	EventName string    `json:"event_name"`
}

type SessionInfo struct {
	EventID   string    `json:"event_id"`
	EventSlug string    `json:"event_slug"`
	EventName string    `json:"event_name"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Status struct {
	Configured bool       `json:"configured"`
	Enabled    bool       `json:"enabled"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

type Service interface {
	Status(c *gin.Context, eventID string) (*Status, error)
	SetPassword(c *gin.Context, eventID, password string) (*Status, error)
	Disable(c *gin.Context, eventID string) error
	Login(c *gin.Context, slug, password string) (*LoginResult, error)
	Session(c *gin.Context, slug, token string) (*SessionInfo, *Session, error)
	Tickets(c *gin.Context, slug, token, search, ticketID string) ([]*TicketRow, error)
	TicketOptions(c *gin.Context, slug, token string) ([]*TicketOption, error)
	Distribution(c *gin.Context, slug, token string) (*Distribution, error)
	CheckIn(c *gin.Context, slug, token, publicID string) (*presence.ResScanTicket, error)
}

type Handler interface {
	AdminStatus(c *gin.Context)
	SetPassword(c *gin.Context)
	Disable(c *gin.Context)
	Login(c *gin.Context)
	Session(c *gin.Context)
	Tickets(c *gin.Context)
	TicketOptions(c *gin.Context)
	Distribution(c *gin.Context)
	CheckIn(c *gin.Context)
}
