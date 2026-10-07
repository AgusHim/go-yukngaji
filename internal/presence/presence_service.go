package presence

import (
	"log"
	"mainyuk/internal/apperr"
	"mainyuk/internal/event"
	"mainyuk/internal/order"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// checkInGracePeriod adalah toleransi setelah event berakhir; pemindaian masih
// diterima selama rentang ini supaya check-in di hari terakhir tidak gagal
// karena acara sedikit melewati jadwal.
const checkInGracePeriod = 24 * time.Hour

type service struct {
	PresenceRepository Repository
	UserService        user.Service
	EventService       event.Service
	UserTicketService  user_ticket.Service
	checkInRewarder    CheckInRewarder
}

func NewService(repository Repository, userService user.Service, eventService event.Service, userTicketService user_ticket.Service) Service {
	return &service{
		PresenceRepository: repository,
		UserService:        userService,
		EventService:       eventService,
		UserTicketService:  userTicketService,
	}
}

// SetCheckInRewarder memasang pemberi reward check-in.
func (s *service) SetCheckInRewarder(r CheckInRewarder) {
	s.checkInRewarder = r
}

// rewardCheckIn memberi XP check-in kepada peserta terverifikasi.
//
// Pemilik reward adalah ticket.ParticipantUserID secara langsung. Bila nil
// (tiket legacy atau peserta belum diklaim), tidak ada XP: sengaja tidak
// memakai ParticipantUserIDValue(), karena fallback ke pembeli tiket akan
// memberi XP kepada orang yang tidak menghadiri acara.
//
// Best-effort: kegagalan dicatat, bukan dikembalikan, supaya pemindaian tiket
// tidak ikut gagal karena masalah XP. Pemindaian ulang akan mencoba lagi.
func (s *service) rewardCheckIn(c *gin.Context, ticket *user_ticket.UserTicket) {
	if s.checkInRewarder == nil || ticket == nil || ticket.ParticipantUserID == nil {
		return
	}
	if *ticket.ParticipantUserID == "" {
		return
	}
	if err := s.checkInRewarder.OnVerifiedCheckIn(c, *ticket.ParticipantUserID, ticket.EventID); err != nil {
		log.Printf("[xp] gagal memberi reward check-in untuk tiket %s: %v", ticket.ID, err)
	}
}

// Create mencatat kehadiran peserta pada sebuah event.
//
// Bila pemanggil login, identitas diambil dari server dan `user_id` di body
// diabaikan. Jalur tamu tetap didukung: data user di body dipakai untuk
// membuat akun `jamaah`.
func (s *service) Create(c *gin.Context, req *CreatePresence) (*Presence, error) {
	currentUser, loggedIn := user.FromContext(c)
	if loggedIn {
		userID := currentUser.ID
		req.UserID = &userID
		req.User = nil
	}

	if req.User == nil && req.UserID == nil {
		return nil, apperr.ErrInvalidRequest
	}

	event, errEvent := s.EventService.Show(c, req.EventID)
	if errEvent != nil {
		return nil, ErrEventNotFound
	}

	if req.UserID != nil {
		presence, _ := s.PresenceRepository.FindByUserID(c, *req.UserID, event.ID)
		if presence != nil {
			return presence, nil
		}
	}

	now := time.Now()
	if event.CloseAt != nil {
		if now.After(*event.CloseAt) {
			return nil, ErrRegisterClosed
		}
	}
	if now.After(event.EndAt) {
		return nil, ErrEventClosed
	}

	presence := &Presence{}
	presence.Event = event
	presence.ID = uuid.NewString()
	presence.EventID = event.ID

	var participant *user.User
	if req.UserID == nil && req.User != nil {
		u, err := s.UserService.Presence(c, req.User)
		if err != nil {
			return nil, err
		}
		presence.UserID = u.ID
		participant = u
	}
	if req.UserID != nil && req.User == nil {
		u, errUser := s.UserService.Show(c, *req.UserID)
		if errUser != nil {
			return nil, ErrParticipantAbsent
		}
		presence.UserID = u.ID
		participant = u
	}

	presence.User = participant
	presence.CreatedAt = time.Now()
	presence.UpdatedAt = time.Now()

	presence, err := s.PresenceRepository.Create(c, presence)
	if err != nil {
		// Hapus akun yang baru dibuat supaya tidak meninggalkan akun yatim.
		if participant != nil && req.User != nil {
			if err := s.UserService.DeleteByID(c, participant.ID); err != nil {
				return nil, err
			}
		}
		return nil, err
	}

	// Kenaikan penghitung dilakukan di database supaya dua pendaftaran
	// bersamaan tidak saling menimpa.
	if errIncrement := s.EventService.IncrementParticipant(c, event.ID); errIncrement != nil {
		return nil, errIncrement
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

// CreateFromTicket memproses pemindaian tiket oleh ranger.
//
// Aturan yang ditegakkan server:
//   - tiket harus milik event yang di-scan,
//   - order tiket harus sudah `paid`,
//   - event belum lewat (plus masa tenggang).
//
// Check-in bersifat idempotent per hari (Asia/Jakarta): pemindaian ulang pada
// hari yang sama mengembalikan hasil yang sudah ada sebagai sukses, dan
// penghitung peserta hanya naik pada check-in pertama tiket.
func (s *service) CreateFromTicket(c *gin.Context, slug string, public_id string) (*ResScanTicket, error) {
	event, errEvent := s.EventService.Show(c, slug)
	if errEvent != nil {
		return nil, ErrEventNotFound
	}

	ticket, errTicket := s.UserTicketService.ShowByPublicID(c, public_id)
	if errTicket != nil {
		return nil, ErrTicketNotFound
	}

	// Tiket harus milik event yang sedang di-scan.
	if ticket.EventID != event.ID {
		return nil, ErrTicketWrongEvent
	}

	// Hanya tiket dari order yang sudah dibayar yang boleh masuk.
	if ticket.Order == nil {
		return nil, ErrOrderNotFound
	}
	if ticket.Order.Status != order.StatusPaid {
		return nil, ErrTicketNotPaid
	}

	now := time.Now()
	checkInDate := CheckInDate(now, JakartaLocation)
	dateLabel := checkInDate.Format("2006-01-02")

	// Event yang sudah lama berakhir tidak boleh lagi di-scan.
	if !event.EndAt.IsZero() && now.After(event.EndAt.Add(checkInGracePeriod)) {
		return nil, ErrEventClosed
	}

	// Jalur cepat: sudah check-in hari ini.
	if existing, errFind := s.PresenceRepository.FindByUserTicketAndDate(c, ticket.ID, checkInDate); errFind == nil && existing != nil {
		return s.scanResult(c, ticket, true, dateLabel)
	}

	adminID, errAdminID := s.GetUserIDAuth(c)
	if errAdminID != nil {
		return nil, errAdminID
	}

	// Check-in pertama tiket ini? Hanya itu yang menaikkan penghitung peserta.
	priorCount, errCount := s.PresenceRepository.CountByUserTicket(c, ticket.ID)
	if errCount != nil {
		return nil, errCount
	}

	// Peserta rombongan memakai akun peserta; baris lama tanpa
	// participant_user_id jatuh kembali ke pembeli.
	participantID := ticket.ParticipantUserIDValue()
	participant, errUser := s.UserService.Show(c, participantID)
	if errUser != nil {
		return nil, ErrParticipantAbsent
	}

	presence := &Presence{}
	userTicketID := ticket.ID
	presence.UserTicketID = &userTicketID
	presence.UserTicket = *ticket
	presence.ID = uuid.NewString()
	presence.EventID = event.ID
	presence.Event = event
	presence.UserID = participant.ID
	presence.User = participant
	presence.AdminID = &adminID
	presence.CheckInDate = &checkInDate
	presence.CreatedAt = now
	presence.UpdatedAt = now

	if _, err := s.PresenceRepository.Create(c, presence); err != nil {
		// Pemindaian paralel bisa sama-sama lolos pengecekan di atas; yang
		// kalah balapan tetap dianggap sukses bila barisnya sudah ada.
		if existing, errFind := s.PresenceRepository.FindByUserTicketAndDate(c, ticket.ID, checkInDate); errFind == nil && existing != nil {
			return s.scanResult(c, ticket, true, dateLabel)
		}
		return nil, err
	}

	if priorCount == 0 {
		if errIncrement := s.EventService.IncrementParticipant(c, event.ID); errIncrement != nil {
			return nil, errIncrement
		}
	}

	return s.scanResult(c, ticket, false, dateLabel)
}

func (s *service) scanResult(c *gin.Context, ticket *user_ticket.UserTicket, alreadyCheckedIn bool, dateLabel string) (*ResScanTicket, error) {
	// Dipanggil di satu-satunya titik keluar pemindaian yang berhasil, jadi
	// baik check-in baru maupun pemindaian ulang sama-sama memicu hook ini.
	// Dedup di sisi ledger yang memastikan rewardnya tetap sekali per event.
	s.rewardCheckIn(c, ticket)

	history, err := s.PresenceRepository.IndexByUserTicket(c, ticket.ID)
	if err != nil {
		return nil, ErrPresenceNotFound
	}
	res := &ResScanTicket{}
	res.UserTicket = *ticket
	res.Presences = presenceToPresences(history)
	res.AlreadyCheckedIn = alreadyCheckedIn
	res.CheckInDate = dateLabel
	return res, nil
}

func (s *service) IndexByUserTicket(c *gin.Context, ut_id string) ([]*Presence, error) {
	presence, err := s.PresenceRepository.IndexByUserTicket(c, ut_id)
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (s *service) GetUserIDAuth(c *gin.Context) (string, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return "", apperr.ErrUnauthorized
	}
	return currentUser.ID, nil
}
