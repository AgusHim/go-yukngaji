package event_verifier

import (
	context "context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"mainyuk/internal/presence"
	"mainyuk/internal/ratelimit"
	"mainyuk/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrUnauthorized = errors.New("password atau sesi verifikator tidak valid")
	ErrDisabled     = errors.New("akses verifikator event belum aktif")
	ErrInvalid      = errors.New("password verifikator minimal 8 karakter")
)

type service struct {
	db       *gorm.DB
	presence presence.Service
}

func NewService(db *gorm.DB, presenceService presence.Service) Service {
	return &service{db: db, presence: presenceService}
}

func requestContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

func (s *service) Status(c *gin.Context, eventID string) (*Status, error) {
	var value Credential
	err := s.db.WithContext(requestContext(c)).Where("event_id = ?", eventID).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &Status{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &Status{Configured: true, Enabled: value.Enabled, UpdatedAt: &value.UpdatedAt}, nil
}

func (s *service) SetPassword(c *gin.Context, eventID, password string) (*Status, error) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return nil, ErrInvalid
	}
	var count int64
	if err := s.db.WithContext(requestContext(c)).Table("events").Where("id = ? AND deleted_at IS NULL", eventID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	value := Credential{EventID: eventID, PasswordHash: hash, Enabled: true, CreatedAt: now, UpdatedAt: now}
	err = s.db.WithContext(requestContext(c)).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoUpdates: clause.Assignments(map[string]any{"password_hash": hash, "enabled": true, "updated_at": now}),
		}).Create(&value).Error; err != nil {
			return err
		}
		return tx.Model(&Session{}).Where("event_id = ? AND revoked_at IS NULL", eventID).Update("revoked_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Status(c, eventID)
}

func (s *service) Disable(c *gin.Context, eventID string) error {
	return s.db.WithContext(requestContext(c)).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Credential{}).Where("event_id = ?", eventID).Updates(map[string]any{"enabled": false, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&Session{}).Where("event_id = ? AND revoked_at IS NULL", eventID).Update("revoked_at", now).Error
	})
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := cryptorand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func bearer(c *gin.Context) string {
	parts := strings.SplitN(strings.TrimSpace(c.GetHeader("Authorization")), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (s *service) Login(c *gin.Context, slug, password string) (*LoginResult, error) {
	key := "verifier:" + slug + ":" + ratelimit.ClientIP(c)
	if !ratelimit.VerifierLogin.Allow(key) {
		return nil, ratelimit.ErrTooManyRequests
	}
	var row struct {
		EventID, Slug, Title, PasswordHash string
		Enabled                            bool
	}
	err := s.db.WithContext(requestContext(c)).Table("events").
		Select("events.id AS event_id, events.slug, events.title, credential.password_hash, credential.enabled").
		Joins("JOIN event_verifier_credentials credential ON credential.event_id = events.id").
		// Take avoids GORM inferring `event_id` as the primary key of the
		// projection struct and emitting `ORDER BY events.event_id`. The events
		// table's key is `id`; event_id only exists on the verifier tables.
		Where("events.slug = ? AND events.deleted_at IS NULL", slug).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDisabled
	}
	if err != nil {
		return nil, err
	}
	if !row.Enabled {
		return nil, ErrDisabled
	}
	if err := utils.CheckPassword(password, row.PasswordHash); err != nil {
		return nil, ErrUnauthorized
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	session := Session{ID: uuid.NewString(), EventID: row.EventID, TokenHash: tokenHash(token), ExpiresAt: now.Add(sessionLifetime), LastUsedAt: now, CreatedAt: now}
	if err := s.db.WithContext(requestContext(c)).Create(&session).Error; err != nil {
		return nil, err
	}
	ratelimit.VerifierLogin.Reset(key)
	return &LoginResult{Token: token, ExpiresAt: session.ExpiresAt, EventID: row.EventID, EventSlug: row.Slug, EventName: row.Title}, nil
}

func (s *service) Session(c *gin.Context, slug, token string) (*SessionInfo, *Session, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil, ErrUnauthorized
	}
	var row struct {
		SessionID string
		EventID   string
		Slug      string
		Title     string
		ExpiresAt time.Time
	}
	err := s.db.WithContext(requestContext(c)).Table("event_verifier_sessions session").
		Select("session.id AS session_id, session.event_id, events.slug, events.title, session.expires_at").
		Joins("JOIN events ON events.id = session.event_id").
		Joins("JOIN event_verifier_credentials credential ON credential.event_id = session.event_id").
		Where("session.token_hash = ? AND events.slug = ? AND session.expires_at > ? AND session.revoked_at IS NULL AND credential.enabled = true", tokenHash(token), slug, time.Now().UTC()).
		// The projection's SessionID is an alias, not a column on `session`.
		// Take prevents GORM from trying to order by session.session_id.
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrUnauthorized
	}
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	s.db.WithContext(requestContext(c)).Model(&Session{}).Where("id = ?", row.SessionID).Update("last_used_at", now)
	session := &Session{ID: row.SessionID, EventID: row.EventID, ExpiresAt: row.ExpiresAt}
	return &SessionInfo{EventID: row.EventID, EventSlug: row.Slug, EventName: row.Title, ExpiresAt: row.ExpiresAt}, session, nil
}

func (s *service) Tickets(c *gin.Context, slug, token, search, ticketID string) ([]*TicketRow, error) {
	info, _, err := s.Session(c, slug, token)
	if err != nil {
		return nil, err
	}
	checkDate := presence.CheckInDate(time.Now(), presence.JakartaLocation)
	query := s.db.WithContext(requestContext(c)).Table("user_tickets ut").
		Select(`ut.public_id, ut.user_name AS name, ut.user_email AS email,
			COALESCE(participant.phone, buyer.phone, '') AS phone, ut.user_gender AS gender,
			tickets.name AS ticket_name,
			EXISTS (SELECT 1 FROM presence p WHERE p.user_ticket_id = ut.id AND p.check_in_date = ? AND p.deleted_at IS NULL) AS checked_in,
			(SELECT MAX(p.created_at) FROM presence p WHERE p.user_ticket_id = ut.id AND p.deleted_at IS NULL) AS last_check_in_at`, checkDate).
		Joins("JOIN orders ON orders.id = ut.order_id AND orders.status = 'paid'").
		Joins("JOIN tickets ON tickets.id = ut.ticket_id").
		Joins("LEFT JOIN users participant ON participant.id = ut.participant_user_id").
		Joins("LEFT JOIN users buyer ON buyer.id = ut.user_id").
		Where("ut.event_id = ? AND ut.deleted_at IS NULL", info.EventID)
	if ticketID = strings.TrimSpace(ticketID); ticketID != "" {
		query = query.Where("ut.ticket_id = ?", ticketID)
	}
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(ut.public_id) LIKE ? OR LOWER(ut.user_name) LIKE ? OR LOWER(ut.user_email) LIKE ? OR LOWER(COALESCE(participant.phone, buyer.phone, '')) LIKE ?", like, like, like, like)
	}
	var rows []*TicketRow
	if err := query.Order("ut.user_name ASC").Limit(100).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *service) TicketOptions(c *gin.Context, slug, token string) ([]*TicketOption, error) {
	info, _, err := s.Session(c, slug, token)
	if err != nil {
		return nil, err
	}
	var options []*TicketOption
	err = s.db.WithContext(requestContext(c)).Table("tickets").
		Select("tickets.id, tickets.name").
		Joins("JOIN user_tickets ut ON ut.ticket_id = tickets.id AND ut.deleted_at IS NULL").
		Joins("JOIN orders ON orders.id = ut.order_id AND orders.status = 'paid'").
		Where("tickets.event_id = ? AND tickets.deleted_at IS NULL", info.EventID).
		Group("tickets.id, tickets.name").
		Order("tickets.name ASC").
		Scan(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

func (s *service) CheckIn(c *gin.Context, slug, token, publicID string) (*presence.ResScanTicket, error) {
	_, session, err := s.Session(c, slug, token)
	if err != nil {
		return nil, err
	}
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil, fmt.Errorf("kode tiket wajib diisi")
	}
	return s.presence.CreateFromTicketByVerifier(c, slug, publicID, session.ID)
}

func TokenFromRequest(c *gin.Context) string { return bearer(c) }
