package order

import (
	"log"

	"mainyuk/internal/apperr"
	"mainyuk/internal/event"
	"mainyuk/internal/payment_method"
	"mainyuk/internal/ticket"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository
	TicketService     ticket.Service
	UserTicketService user_ticket.Service
	EventService      event.Service
	PaymentMethod     payment_method.Service
	UserService       user.Service
	// poster boleh nil; nil berarti aktivitas pendaftaran tidak dibagikan ke
	// feed, seperti sebelum Fase 3.
	poster AutoPoster
}

// SetAutoPoster memasang peniti aktivitas ke feed komunitas.
func (s *service) SetAutoPoster(poster AutoPoster) {
	s.poster = poster
}

// postRegistration menitipkan pendaftaran ke feed.
//
// Kegagalannya ditelan dan dilaporkan ke log: tiket yang sudah sah tidak boleh
// dibatalkan hanya karena feed gagal. Pemanggilnya sengaja dipanggil ketika
// status akhirnya sudah benar, bukan hanya saat barisnya baru berpindah —
// index unik sumber membuat percobaan ulang aman dan justru memulihkan
// postingan yang tertinggal.
func (s *service) postRegistration(c *gin.Context, userID, orderID, eventTitle string) {
	if s.poster == nil {
		return
	}
	if err := s.poster.PostEventRegistration(c, userID, orderID, eventTitle); err != nil {
		log.Printf("[order] gagal membagikan pendaftaran %s ke feed: %v", orderID, err)
	}
}

// removeRegistration mencabut postingan pendaftaran ketika ordernya tidak jadi
// dibayar. Kegagalannya juga ditelan.
func (s *service) removeRegistration(c *gin.Context, orderID string) {
	if s.poster == nil {
		return
	}
	if err := s.poster.RemoveEventRegistration(c, orderID); err != nil {
		log.Printf("[order] gagal mencabut pendaftaran %s dari feed: %v", orderID, err)
	}
}

func NewService(repository Repository, ticketService ticket.Service, userTicketService user_ticket.Service, eventService event.Service, pmService payment_method.Service, userService user.Service) Service {
	return &service{
		Repository:        repository,
		TicketService:     ticketService,
		UserTicketService: userTicketService,
		EventService:      eventService,
		PaymentMethod:     pmService,
		UserService:       userService,
	}
}

// Create membangun order beserta tiketnya.
//
// Identitas pembeli selalu diambil dari server (currentUser), bukan dari
// body request. Order dan tiketnya disimpan dalam satu transaksi.
func (s *service) Create(c *gin.Context, req *CreateOrder) (*Order, error) {
	event, err := s.EventService.Show(c, req.EventID)
	if err != nil {
		return nil, invalid("event not found")
	}

	currentUser, err := s.GetUserIDAuth(c)
	if err != nil {
		return nil, err
	}

	if len(req.UserTickets) == 0 {
		return nil, invalid("order harus berisi minimal satu tiket")
	}

	now := time.Now()

	// Ambil tiket unik yang dipesan, lalu validasi relasinya ke event.
	byID := make(map[string]*ticket.Ticket, len(req.UserTickets))
	uniqueIDs := make([]string, 0, len(req.UserTickets))
	for _, ut := range req.UserTickets {
		if _, seen := byID[ut.TicketID]; seen {
			continue
		}
		t, errTicket := s.TicketService.Show(c, ut.TicketID)
		if errTicket != nil {
			return nil, invalid("tiket %s tidak ditemukan", ut.TicketID)
		}
		byID[ut.TicketID] = t
		uniqueIDs = append(uniqueIDs, ut.TicketID)
	}

	if err := ValidateTicketEvents(event.ID, req.UserTickets, byID); err != nil {
		return nil, err
	}

	requested := make(map[string]int, len(uniqueIDs))
	for _, ut := range req.UserTickets {
		requested[ut.TicketID]++
	}

	for _, id := range uniqueIDs {
		t := byID[id]
		if err := ValidateTicketWindow(t, now); err != nil {
			return nil, err
		}
		// Kuota hanya diperiksa bila tiket memang membatasinya. Tiket lama
		// dengan max_pax 0 dianggap tidak berkuota.
		if t.MaxPax > 0 {
			sold, errCount := s.UserTicketService.CountByTicketID(c, id)
			if errCount != nil {
				return nil, errCount
			}
			if int(sold)+requested[id] > t.MaxPax {
				return nil, invalid("kuota tiket %s tidak mencukupi", t.Name)
			}
		}
	}

	prices := make([]int, 0, len(req.UserTickets))
	for _, ut := range req.UserTickets {
		prices = append(prices, byID[ut.TicketID].Price)
	}

	order := &Order{}
	order.ID = uuid.NewString()
	order.PublicID = strings.ToUpper(strings.Split(uuid.NewString(), "-")[0])
	order.EventID = event.ID
	order.Event = event
	order.UserID = currentUser.ID
	order.User = currentUser

	if req.PaymentMethodID != "" {
		paymentMethod, errPM := s.PaymentMethod.Show(c, req.PaymentMethodID)
		if errPM != nil {
			return nil, invalid("payment method not found")
		}
		order.PaymentMethodID = &paymentMethod.ID
		order.PaymentMethod = paymentMethod
	}

	donation := 0
	if req.Donation != nil {
		donation = *req.Donation
	}
	adminFee := 0
	if req.AdminFee != nil {
		adminFee = *req.AdminFee
	}

	amounts := ComputeOrderAmounts(prices, donation, adminFee)
	// orders.amount tetap berarti subtotal tiket; donasi dan biaya admin
	// tersimpan di kolomnya sendiri sehingga total selalu bisa dihitung ulang.
	order.Amount = amounts.TicketTotal
	order.Donation = amounts.Donation
	order.AdminFee = amounts.AdminFee
	order.Status = DeriveStatus(amounts.Total)
	order.ComputeTotal()

	order.CreatedAt = now
	order.UpdatedAt = now

	// Kaitkan tiap peserta ke akun yang sudah ada berdasarkan email.
	// Email yang belum terdaftar dibiarkan kosong dan bisa diklaim nanti.
	buyerEmail := ""
	if currentUser.Email != nil {
		buyerEmail = strings.ToLower(strings.TrimSpace(*currentUser.Email))
	}
	participants := MatchParticipants(NormalizeParticipantEmails(req.UserTickets), func(email string) (string, bool) {
		if buyerEmail != "" && email == buyerEmail {
			return currentUser.ID, true
		}
		u, errUser := s.UserService.GetUserByEmail(c, email)
		if errUser != nil || u == nil {
			return "", false
		}
		return u.ID, true
	})

	tickets := make([]*user_ticket.UserTicket, 0, len(req.UserTickets))
	for _, ut := range req.UserTickets {
		email := strings.ToLower(strings.TrimSpace(ut.UserEmail))
		tickets = append(tickets, user_ticket.NewUserTicket(user_ticket.CreateUserTicket{
			UserName:          ut.UserName,
			UserEmail:         ut.UserEmail,
			UserGender:        ut.UserGender,
			UserID:            currentUser.ID,
			OrderID:           order.ID,
			TicketID:          ut.TicketID,
			EventID:           event.ID,
			ParticipantUserID: participants[email],
		}))
	}

	order, err = s.Repository.CreateWithTickets(c, order, tickets)
	if err != nil {
		return nil, err
	}

	order.UserTickets = tickets
	order.ComputeTotal()

	// Order gratis langsung berstatus paid saat dibuat; order berbayar
	// menyusul lewat VerifyOrder.
	if order.Status == StatusPaid {
		s.postRegistration(c, currentUser.ID, order.ID, event.Title)
	}
	return order, nil
}

func (s *service) Show(c *gin.Context, id string) (*Order, error) {
	order, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	tickets, err := s.UserTicketService.IndexByOrderID(c, order.ID)
	if err != nil {
		return nil, err
	}
	order.UserTickets = tickets
	order.ComputeTotal()
	return order, nil
}

func (s *service) ShowByPublicID(c *gin.Context, public_id string) (*Order, error) {
	user, errUserID := s.GetUserIDAuth(c)
	if errUserID != nil {
		return nil, errUserID
	}
	order, err := s.Repository.ShowByPublicID(c, public_id, &user.ID)
	if err != nil {
		return nil, err
	}
	tickets, err := s.UserTicketService.IndexByOrderID(c, order.ID)
	if err != nil {
		return nil, err
	}
	order.UserTickets = tickets
	order.ComputeTotal()
	return order, nil
}

func (s *service) Index(c *gin.Context) ([]*Order, error) {
	user, errUserID := s.GetUserIDAuth(c)
	if errUserID != nil {
		return nil, errUserID
	}
	order, err := s.Repository.Index(c, &user.ID)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) IndexAdmin(c *gin.Context) ([]*Order, error) {
	order, err := s.Repository.Index(c, nil)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) GetUserIDAuth(c *gin.Context) (*user.User, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	return currentUser, nil
}

// VerifyOrder memindahkan status pembayaran mengikuti tabel transisi yang
// sah. Status arbitrer dari client tidak lagi diterima apa adanya.
func (s *service) VerifyOrder(c *gin.Context, id string, status string) (*Order, error) {
	target := strings.ToLower(strings.TrimSpace(status))
	if !IsValidStatus(target) {
		return nil, invalid("status %q tidak dikenal", status)
	}

	order, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}

	if !CanTransition(order.Status, target) {
		return nil, invalid("status tidak bisa berpindah dari %s ke %s", order.Status, target)
	}

	// Perpindahan dilakukan dengan syarat status lama masih berlaku, sehingga
	// dua verifikasi bersamaan tidak bisa menimpa satu sama lain.
	moved, errMove := s.Repository.TransitionStatus(c, order.ID, order.Status, target)
	if errMove != nil {
		return nil, errMove
	}
	if !moved {
		return nil, invalid("status order sudah berubah, muat ulang datanya")
	}

	order.Status = target
	order.UpdatedAt = time.Now()
	order.ComputeTotal()

	// Dipanggil atas dasar status akhirnya, bukan atas dasar "barisnya baru
	// berpindah". Bila proses mati setelah transisi tetapi sebelum feed
	// sempat ditulis, verifikasi ulang tetap menghasilkan postingannya —
	// tepat sekali, karena dedup ada di index unik sumbernya.
	switch target {
	case StatusPaid:
		s.postRegistration(c, order.UserID, order.ID, s.eventTitle(c, order.EventID))
	default:
		// Order yang batal, kedaluwarsa, gagal, atau dana dikembalikan tidak
		// boleh meninggalkan postingan pendaftaran di feed.
		s.removeRegistration(c, order.ID)
	}
	return order, nil
}

// eventTitle mengambil judul event untuk judul postingan. Kegagalan
// pengambilannya tidak menggagalkan verifikasi order; postingannya saja yang
// tidak dibuat, karena judul kosong ditolak ShouldAutoPost.
func (s *service) eventTitle(c *gin.Context, eventID string) string {
	ev, err := s.EventService.Show(c, eventID)
	if err != nil || ev == nil {
		return ""
	}
	return ev.Title
}

func (s *service) Participants(c *gin.Context, event_id string) ([]*Order, error) {
	order, err := s.Repository.Participants(c, event_id)
	if err != nil {
		return nil, err
	}
	return order, nil
}
