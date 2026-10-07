package presence

import (
	"mainyuk/internal/event"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"time"

	"github.com/gin-gonic/gin"
)

type Presence struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"-" binding:"required"`
	EventID      string                 `json:"-"  binding:"required"`
	User         *user.User             `json:"user"`
	Event        *event.Event           `json:"event"`
	UserTicketID *string                `json:"-"`
	UserTicket   user_ticket.UserTicket `json:"user_ticket" gorm:"foreignKey:user_ticket_id;references:id"`
	AdminID      *string                `json:"admin_id"`
	// CheckInDate adalah tanggal check-in (Asia/Jakarta) tanpa jam. Dipakai
	// bersama unique index supaya satu tiket hanya bisa check-in sekali per hari.
	CheckInDate *time.Time `json:"check_in_date" gorm:"column:check_in_date;type:date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `json:"-"`
}

type ResScanTicket struct {
	UserTicket user_ticket.UserTicket `json:"user_ticket" gorm:"foreignKey:ticket_id;references:id"`
	Presences  []*time.Time           `json:"presences"`
	// AlreadyCheckedIn true bila tiket ini sudah check-in pada tanggal yang sama.
	AlreadyCheckedIn bool `json:"already_checked_in"`
	// CheckInDate tanggal check-in hari ini (format YYYY-MM-DD, Asia/Jakarta).
	CheckInDate string `json:"check_in_date"`
}

func presenceToPresences(presences []*Presence) (res []*time.Time) {
	var filtered []*time.Time
	for _, p := range presences {
		filtered = append(filtered, &p.CreatedAt)
	}
	return filtered
}

func (Presence) TableName() string {
	return "presence"
}

type CreatePresence struct {
	EventID      string           `json:"event_id" binding:"required"`
	UserID       *string          `json:"user_id" `
	User         *user.CreateUser `json:"user"`
	UserTicketID *string          `json:"user_ticket_id"`
}

type PresenceFromTicket struct {
	PublicID string `json:"public_id" binding:"required"`
}

// CheckInRewarder memberi XP untuk check-in terverifikasi. Diimplementasikan
// oleh gamification.Service. Dipisah sebagai interface agar paket presence
// tidak perlu mengimpor gamification.
//
// Implementasinya wajib idempoten: pemindaian ulang tetap memanggil hook ini
// supaya kegagalan sesaat dapat pulih, tetapi hanya pemberian pertama yang
// menghasilkan XP.
type CheckInRewarder interface {
	OnVerifiedCheckIn(c *gin.Context, participantUserID, eventID string) error
}

type Repository interface {
	Create(ctx *gin.Context, event *Presence) (*Presence, error)
	Show(ctx *gin.Context, id string) (*Presence, error)
	Index(ctx *gin.Context) ([]*Presence, error)
	IndexByUserTicket(ctx *gin.Context, user_ticket_id string) ([]*Presence, error)
	FindByUserID(ctx *gin.Context, id string, eventID string) (*Presence, error)
	FindByUserTicketID(ctx *gin.Context, id string, eventID string) (*Presence, error)
	// FindByUserTicketAndDate mencari check-in tiket pada tanggal tertentu.
	FindByUserTicketAndDate(ctx *gin.Context, userTicketID string, checkInDate time.Time) (*Presence, error)
	// CountByUserTicket menghitung seluruh check-in tiket (semua hari).
	CountByUserTicket(ctx *gin.Context, userTicketID string) (int64, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreatePresence) (*Presence, error)
	CreateFromTicket(ctx *gin.Context, slug string, public_id string) (*ResScanTicket, error)
	Show(ctx *gin.Context, id string) (*Presence, error)
	Index(ctx *gin.Context) ([]*Presence, error)
	// SetCheckInRewarder memasang pemberi reward check-in. Best-effort:
	// kegagalan pemberian XP tidak boleh menggagalkan pemindaian tiket.
	SetCheckInRewarder(r CheckInRewarder)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
	CreateFromTicket(ctx *gin.Context)
}
