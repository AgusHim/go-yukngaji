package main

import (
	"context"
	"fmt"
	"log"
	"mainyuk/db"
	"mainyuk/internal/agenda"
	"mainyuk/internal/auth"
	"mainyuk/internal/comment"
	"mainyuk/internal/community"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/feedback"
	"mainyuk/internal/fundraising"
	"mainyuk/internal/gamification"
	"mainyuk/internal/instagram"
	"mainyuk/internal/like"
	"mainyuk/internal/metrics"
	"mainyuk/internal/mission"
	"mainyuk/internal/order"
	"mainyuk/internal/otp"
	"mainyuk/internal/payment"
	"mainyuk/internal/payment_method"
	"mainyuk/internal/poll"
	"mainyuk/internal/presence"
	"mainyuk/internal/ranger"
	"mainyuk/internal/ranger_presence"
	"mainyuk/internal/region"
	"mainyuk/internal/shop"
	"mainyuk/internal/shopee"
	"mainyuk/internal/shopee_qris"
	"mainyuk/internal/thread"
	"mainyuk/internal/ticket"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"mainyuk/internal/ws"
	"mainyuk/router"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// .env bersifat opsional: di container variabel datang dari environment
	// `docker run`. Dulu kegagalan memuat .env langsung menghentikan proses,
	// sehingga image tidak bisa dijalankan tanpa membakar berkas rahasia.
	if errEnv := godotenv.Load(); errEnv != nil {
		log.Printf("Info: .env tidak dimuat (%v); memakai environment yang ada", errEnv)
	}

	db, err := db.NewDatabase()
	if err != nil {
		// Fatalf harus di goroutine utama supaya proses benar-benar berhenti.
		log.Fatalf("Could not initialize DB Connection: %s", err)
	}

	hub := ws.NewHub()

	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository)
	userHandler := user.NewHandler(userService)

	eventRepository := event.NewRepository(db)
	eventService := event.NewService(eventRepository)
	eventHandler := event.NewHandler(eventService)

	divisiRepository := divisi.NewRepository(db)
	divisiService := divisi.NewService(divisiRepository)
	divisiHandler := divisi.NewHandler(divisiService)

	authMiddleware := auth.NewMiddleware(userService)

	// Handler WebSocket memakai middleware auth untuk memvalidasi token yang
	// dikirim lewat query, jadi dibuat setelah middleware tersedia.
	wsHandler := ws.NewHandler(hub, authMiddleware)

	commentRepository := comment.NewRepository(db)
	commentService := comment.NewService(commentRepository, userService, eventService, hub)
	commentHandler := comment.NewHandler(commentService)

	likeRepository := like.NewRepository(db)
	likeService := like.NewService(likeRepository, userService, commentService, hub)
	likeHandler := like.NewHandler(likeService)

	feedbackRepository := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepository, userService, eventService)
	feedbackHandler := feedback.NewHandler(feedbackService)

	agendaRepository := agenda.NewRepository(db)
	agendaService := agenda.NewService(agendaRepository)
	agendaHandler := agenda.NewHandler(agendaService)

	rangerRepository := ranger.NewRepository(db)
	rangerService := ranger.NewService(rangerRepository, userService, divisiService)
	rangerHandler := ranger.NewHandler(rangerService)

	rangerPresenceRepository := ranger_presence.NewRepository(db)
	rangerPresenceService := ranger_presence.NewService(rangerPresenceRepository, rangerService, agendaService, divisiService)
	rangerPresenceHandler := ranger_presence.NewHandler(rangerPresenceService)

	ticketRepository := ticket.NewRepository(db)
	ticketService := ticket.NewService(ticketRepository)
	ticketHandler := ticket.NewHandler(ticketService)

	userTicketRepository := user_ticket.NewRepository(db)
	userTicketService := user_ticket.NewService(userTicketRepository)
	userTicketHandler := user_ticket.NewHandler(userTicketService)

	// Tiket yang emailnya baru terdaftar sebagai akun diklaim otomatis setelah
	// login/OTP/Google berhasil. Best-effort: kegagalan tidak menggagalkan login.
	userService.SetTicketClaimer(userTicketService)

	paymentMethodRepository := payment_method.NewRepository(db)
	paymentMethodService := payment_method.NewService(paymentMethodRepository)
	paymentMethodHandler := payment_method.NewHandler(paymentMethodService)
	shopeeService := shopee.NewFromEnv()
	shopeeQRISService := shopee_qris.New(db, shopeeService)
	if err := shopeeQRISService.Validate(context.Background()); err != nil {
		log.Fatalf("Shopee QRIS tidak dapat dimulai: %v", err)
	}
	shopeeService.SetPaymentLookup(shopeeQRISService)
	paymentProvider := payment.NewShopeeQRISProvider(paymentMethodService, shopeeQRISService)

	orderRepository := order.NewRepository(db)
	orderService := order.NewService(orderRepository, ticketService, userTicketService, eventService, paymentMethodService, userService, paymentProvider)
	// Perbarui transaksi event yang melewati expired_at, termasuk transaksi
	// manual yang tidak memiliki worker pembayaran sendiri.
	orderService.StartExpiryWorker(context.Background())
	orderHandler := order.NewHandler(orderService)

	regionRepository := region.NewRepository(db)
	regionService := region.NewService(regionRepository)
	regionHandler := region.NewHandler(regionService)

	presenceRepository := presence.NewRepository(db)
	presenceService := presence.NewService(presenceRepository, userService, eventService, userTicketService)
	presenceHandler := presence.NewHandler(presenceService)

	otpRepository := otp.NewRepository(db)
	otpService := otp.NewService(otpRepository, userService)
	otpHandler := otp.NewHandler(otpService)

	pollRepository := poll.NewRepository(db)
	pollService := poll.NewService(pollRepository)
	pollHandler := poll.NewHandler(pollService)

	// Fase 1: profil komunitas -> gamifikasi -> misi.
	//
	// Urutan konstruksi mengikuti arah ketergantungan: gamification memakai
	// community sebagai ProfileEnsurer, dan mission memakai gamification
	// sebagai RewardGranter. Kedua hook dipasang setelah semua service ada,
	// supaya tidak ada ketergantungan melingkar saat konstruksi.
	communityRepository := community.NewRepository(db)
	communityService := community.NewService(communityRepository, userService)
	communityHandler := community.NewHandler(communityService)

	gamificationRepository := gamification.NewRepository(db)
	gamificationService := gamification.NewService(gamificationRepository, communityService, userService)
	gamificationHandler := gamification.NewHandler(gamificationService)

	missionRepository := mission.NewRepository(db)
	missionService := mission.NewService(missionRepository, gamificationService, communityService)
	missionHandler := mission.NewHandler(missionService)

	// Fase 2: fundraising memakai provider gabungan: SHOPEE_QRIS otomatis,
	// metode lain tetap manual. community dipakai sebagai DonorResolver
	// (pembaca profil, termasuk visibilitas dan status blokir), dan
	// gamification sebagai RewardGranter donasi.
	fundraisingRepository := fundraising.NewRepository(db)
	fundraisingService := fundraising.NewService(
		fundraisingRepository,
		communityService,
		gamificationService,
		paymentMethodService,
		paymentProvider,
	)
	fundraisingHandler := fundraising.NewHandler(fundraisingService)

	// Fase 3: threads anonim. Feed REST tanpa realtime — pengamanan WebSocket
	// belum ditinjau, jadi siaran tidak ditambahkan di sini.
	//
	// community dipakai sebagai AuthorResolver sekaligus penegak blokir saat
	// menulis; audit dipakai bersama fundraising lewat paket internal/audit,
	// dan repository thread membuat recordernya sendiri dari *gorm.DB yang sama.
	threadRepository := thread.NewRepository(db)
	threadService := thread.NewService(threadRepository, communityService)
	threadHandler := thread.NewHandler(threadService)

	// Fase 4: marketplace merchandise. Metode manual tetap diverifikasi
	// pengurus, sedangkan SHOPEE_QRIS diselesaikan worker bersama di bawah.
	//
	// Provider-nya adalah adapter yang sama dengan donasi (internal/payment),
	// dan gamification dipakai sebagai RewardGranter pesanan. Repository toko
	// membuat penulis auditnya sendiri dari *gorm.DB yang sama.
	//
	// Tanpa SetAutoPoster: membeli merchandise bukan aktivitas yang layak
	// disiarkan, dan menyiarkannya akan membocorkan perilaku pembelian.
	shopRepository := shop.NewRepository(db)
	shopService := shop.NewService(
		shopRepository,
		gamificationService,
		paymentProvider,
	)
	shopHandler := shop.NewHandler(shopService)

	// Lintas modul: laporan agregat. Modul ini hanya membaca tabel yang sudah
	// ada — tidak ada tabel, migrasi, maupun penulisan baru. Ia tidak
	// bergantung pada service lain, jadi dibangun di akhir.
	metricsRepository := metrics.NewRepository(db)
	metricsService := metrics.NewService(metricsRepository)
	metricsHandler := metrics.NewHandler(metricsService)

	// Dinding komunitas Instagram. Client Graph hanya dibuat bila token dan IG
	// user id tersedia; tanpa keduanya NewGraphClient mengembalikan nil dan
	// sinkronisasi dijawab "belum dikonfigurasi", sementara dinding tetap jalan
	// dengan postingan kurasi admin.
	instagramClient := instagram.NewGraphClient(
		os.Getenv("INSTAGRAM_ACCESS_TOKEN"),
		os.Getenv("INSTAGRAM_IG_USER_ID"),
		os.Getenv("INSTAGRAM_GRAPH_VERSION"),
	)
	instagramRepository := instagram.NewRepository(db)
	instagramService := instagram.NewService(instagramRepository, instagramClient)
	instagramHandler := instagram.NewHandler(instagramService)

	// XP profil diberikan setelah profil tersimpan, XP check-in setelah
	// pemindaian tiket berhasil. Keduanya best-effort.
	userService.SetProfileRewarder(gamificationService)
	presenceService.SetCheckInRewarder(gamificationService)

	// Auto-post aktivitas ke feed. Dipasang setelah threadService ada, dan
	// kegagalannya tidak pernah menggagalkan aksi sumbernya.
	orderService.SetAutoPoster(threadService)
	fundraisingService.SetAutoPoster(threadService)
	missionService.SetAutoPoster(threadService)

	// Akun yang dibatasi tidak boleh menulis di mana pun. Tanpa seam ini,
	// pembatasan bisa dilewati hanya dengan pindah ke QnA event.
	commentService.SetAuthorGuard(communityService)
	likeService.SetAuthorGuard(communityService)

	// Mutasi diklaim oleh shopee_qris_charges sebelum menyentuh domain. Tiap
	// cabang settlement idempoten, sehingga restart di tengah proses aman.
	shopeeQRISService.SetSettler(func(c *gin.Context, entityType, entityID string, paidAmount int64, reference string) error {
		switch entityType {
		case shopee_qris.EntityEventOrder:
			return orderService.ConfirmShopeePayment(c, entityID, paidAmount, reference)
		case shopee_qris.EntityDonation:
			return fundraisingService.ConfirmShopeePayment(c, entityID, paidAmount, reference)
		case shopee_qris.EntityShopOrder:
			return shopService.ConfirmShopeePayment(c, entityID, paidAmount, reference)
		default:
			return fmt.Errorf("jenis entitas QRIS tidak dikenal: %s", entityType)
		}
	})
	shopeeQRISService.SetEligibility(func(c *gin.Context, entityType, entityID string) (bool, error) {
		switch entityType {
		case shopee_qris.EntityEventOrder:
			return orderService.ShopeePaymentPending(c, entityID)
		case shopee_qris.EntityDonation:
			return fundraisingService.ShopeePaymentPending(c, entityID)
		case shopee_qris.EntityShopOrder:
			return shopService.ShopeePaymentPending(c, entityID)
		default:
			return false, fmt.Errorf("jenis entitas QRIS tidak dikenal: %s", entityType)
		}
	})
	shopeeQRISService.Start(context.Background())

	go hub.Run()

	router.InitRouter(
		authMiddleware,
		userHandler,
		eventHandler,
		divisiHandler,
		presenceHandler,
		commentHandler,
		likeHandler,
		feedbackHandler,
		wsHandler,
		agendaHandler,
		rangerHandler,
		rangerPresenceHandler,
		orderHandler,
		ticketHandler,
		userTicketHandler,
		paymentMethodHandler,
		regionHandler,
		otpHandler,
		pollHandler,
		communityHandler,
		gamificationHandler,
		missionHandler,
		fundraisingHandler,
		threadHandler,
		shopHandler,
		metricsHandler,
		instagramHandler,
		shopeeService,
	)

	// output current time zone
	fmt.Print("Local time zone ")
	fmt.Println(time.Now().Zone())
	fmt.Println(time.Now().Format("2006-01-02T15:04:05.000 MST"))

	host := os.Getenv("HOST")
	if err := router.Start(host); err != nil {
		log.Fatalf("HTTP server stopped: %v", err)
	}
}
