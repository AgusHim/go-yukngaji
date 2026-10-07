package router

import (
	"log"
	"mainyuk/internal/agenda"
	"mainyuk/internal/auth"
	"mainyuk/internal/comment"
	"mainyuk/internal/community"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/feedback"
	"mainyuk/internal/fundraising"
	"mainyuk/internal/gamification"
	"mainyuk/internal/like"
	"mainyuk/internal/metrics"
	"mainyuk/internal/mission"
	"mainyuk/internal/order"
	"mainyuk/internal/origins"
	"mainyuk/internal/otp"
	"mainyuk/internal/payment_method"
	"mainyuk/internal/poll"
	"mainyuk/internal/presence"
	"mainyuk/internal/ranger"
	"mainyuk/internal/ranger_presence"
	"mainyuk/internal/region"
	"mainyuk/internal/shop"
	"mainyuk/internal/thread"
	"mainyuk/internal/ticket"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"mainyuk/internal/ws"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var r *gin.Engine

func InitRouter(
	authMiddleware auth.Middleware,
	userHandler user.Handler,
	eventHandler event.Handler,
	divisiHandler divisi.Handler,
	presenceHandler presence.Handler,
	commentHandler comment.Handler,
	likeHandler like.Handler,
	feedbackHandler feedback.Handler,
	wsHandler *ws.Handler,
	agendaHandler agenda.Handler,
	rangerHandler ranger.Handler,
	rangerPresenceHandler ranger_presence.Handler,
	orderHandler order.Handler,
	ticketHandler ticket.Handler,
	userTicketHandler user_ticket.Handler,
	paymentMethodHandler payment_method.Handler,
	regionHandler region.Handler,
	otpHandler otp.Handler,
	pollHandler poll.Handler,
	communityHandler community.Handler,
	gamificationHandler gamification.Handler,
	missionHandler mission.Handler,
	fundraisingHandler fundraising.Handler,
	threadHandler thread.Handler,
	shopHandler shop.Handler,
	metricsHandler metrics.Handler,
) {
	mode := os.Getenv("GIN_MODE")
	gin.SetMode(mode)

	r = gin.Default()
	config := cors.DefaultConfig()
	// Origin dibaca dari CORS_ALLOWED_ORIGINS. "*" tidak lagi dipakai karena
	// kombinasi wildcard + credentials ditolak browser dan membuka API ke
	// situs mana pun.
	config.AllowOrigins = origins.Allowed()
	config.AllowCredentials = origins.AllowsCredentials()
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}
	r.Use(cors.New(config))

	api := r.Group("api")
	user_api := r.Group("user_api")
	ranger_api := r.Group("ranger_api")
	admin_api := r.Group("admin_api")

	api.POST("/register", userHandler.Register)
	api.POST("/login", userHandler.Login)
	api.GET("/auth/google/login", userHandler.AuthGoogleLogin)
	api.GET("/auth/google/callback", userHandler.AuthGoogleCallback)
	api.POST("/auth/otp/request", otpHandler.RequestOTP)
	api.POST("/auth/otp/verify", otpHandler.VerifyOTP)
	user_api.GET("/me", authMiddleware.AuthUser, userHandler.Me)
	user_api.PUT("/auth", authMiddleware.AuthUser, userHandler.UpdateAuth)
	admin_api.PUT("/users/:id", authMiddleware.AuthPJ, userHandler.UpdateByAdmin)
	// Daftar akun berhalaman. Didahulukan dari /users/:id supaya "users" tanpa
	// id tidak pernah ditafsirkan sebagai id.
	admin_api.GET("/users", authMiddleware.AuthPJ, userHandler.List)
	admin_api.GET("/users/:id", authMiddleware.AuthPJ, userHandler.Show)

	api.POST("/events", authMiddleware.AuthAdmin, eventHandler.Create)
	api.GET("/events/:slug", eventHandler.Show)
	api.GET("/events/code/:code", eventHandler.ShowByCode)
	api.GET("/events", eventHandler.Index)
	api.PUT("/events/:id", authMiddleware.AuthAdmin, eventHandler.Create)
	admin_api.GET("/events/:event_id/participants", authMiddleware.AuthAdmin, userTicketHandler.IndexByEventID)

	admin_api.POST("/divisi", authMiddleware.AuthPJ, divisiHandler.Create)
	admin_api.GET("/divisi/:slug", authMiddleware.AuthPJ, divisiHandler.Show)
	admin_api.GET("/divisi", authMiddleware.AuthPJ, divisiHandler.Index)

	// Identitas didahulukan dari token; tamu tetap boleh mendaftar.
	api.POST("/presence", authMiddleware.AuthOptionalUser, presenceHandler.Create)
	api.GET("/presence/:slug", presenceHandler.Show)
	admin_api.GET("/presence", authMiddleware.AuthAdmin, presenceHandler.Index)
	user_api.GET("/presence", authMiddleware.AuthUser, presenceHandler.Index)
	ranger_api.POST("/event/:slug/presence", authMiddleware.AuthRanger, presenceHandler.CreateFromTicket)

	// Menulis komentar/like wajib login; identitas diambil server dari token.
	api.POST("/comments", authMiddleware.AuthUser, commentHandler.Create)
	api.GET("/comments", commentHandler.Index)

	api.GET("/comments/like", likeHandler.Index)
	api.POST("/comments/like", authMiddleware.AuthUser, likeHandler.Create)
	api.DELETE("/comments/like/:id", authMiddleware.AuthUser, likeHandler.Delete)

	api.GET("/feedback", authMiddleware.AuthAdmin, feedbackHandler.Index)
	// feedback.user_id NOT NULL, jadi masukan hanya untuk pengguna login.
	api.POST("/feedback", authMiddleware.AuthUser, feedbackHandler.Create)

	admin_api.POST("/agenda", authMiddleware.AuthPJ, agendaHandler.Create)
	admin_api.GET("/agenda/:id", authMiddleware.AuthPJ, agendaHandler.Show)
	admin_api.GET("/agenda", authMiddleware.AuthPJ, agendaHandler.Index)
	admin_api.PUT("/agenda/:id", authMiddleware.AuthPJ, agendaHandler.Update)
	admin_api.DELETE("/agenda/:id", authMiddleware.AuthPJ, agendaHandler.Delete)

	admin_api.POST("/rangers", authMiddleware.AuthPJ, rangerHandler.Create)
	ranger_api.GET("/rangers/me", authMiddleware.AuthRanger, rangerHandler.Show)
	admin_api.GET("/rangers/:id", authMiddleware.AuthPJ, rangerHandler.Show)
	admin_api.GET("/rangers", authMiddleware.AuthPJ, rangerHandler.Index)
	admin_api.PUT("/rangers/:id", authMiddleware.AuthPJ, rangerHandler.Update)
	admin_api.DELETE("/rangers/:id", authMiddleware.AuthPJ, rangerHandler.Delete)

	admin_api.POST("/rangers/presence", authMiddleware.AuthPJ, rangerPresenceHandler.Create)
	admin_api.GET("/rangers/presence/:id", authMiddleware.AuthPJ, rangerPresenceHandler.Show)

	admin_api.GET("/rangers/presence", authMiddleware.AuthPJ, rangerPresenceHandler.Index)
	ranger_api.GET("/rangers/presence", authMiddleware.AuthRanger, rangerPresenceHandler.Index)

	/* Tickets */
	api.GET("/tickets", ticketHandler.Index)
	admin_api.GET("/tickets", authMiddleware.AuthAdmin, ticketHandler.Index)
	admin_api.POST("/tickets", authMiddleware.AuthAdmin, ticketHandler.Create)
	admin_api.PUT("/tickets/:id", authMiddleware.AuthAdmin, ticketHandler.Update)
	admin_api.DELETE("/tickets/:id", authMiddleware.AuthAdmin, ticketHandler.Delete)

	/* Orders */
	user_api.GET("/orders", authMiddleware.AuthUser, orderHandler.Index)
	user_api.POST("/orders", authMiddleware.AuthUser, orderHandler.Create)
	user_api.GET("/orders/:public_id", authMiddleware.AuthUser, orderHandler.ShowByPublicID)
	admin_api.GET("/orders", authMiddleware.AuthAdmin, orderHandler.IndexAdmin)
	admin_api.GET("/orders/:public_id", authMiddleware.AuthAdmin, orderHandler.ShowByPublicID)
	admin_api.PUT("/orders/:id/verify", authMiddleware.AuthAdmin, orderHandler.VerifyOrder)

	/* Payment Method */
	user_api.GET("/payment_methods", authMiddleware.AuthUser, paymentMethodHandler.Index)
	admin_api.POST("/payment_methods", authMiddleware.AuthAdmin, paymentMethodHandler.Create)
	admin_api.PUT("/payment_methods/:id", authMiddleware.AuthAdmin, paymentMethodHandler.Update)
	admin_api.DELETE("/payment_methods/:id", authMiddleware.AuthAdmin, paymentMethodHandler.Delete)

	/* Region */
	api.GET("/province", regionHandler.Index)
	api.GET("/district", regionHandler.Index)
	api.GET("/sub_district", regionHandler.Index)

	/* User Ticket */
	ranger_api.GET("/user_tickets/:public_id", authMiddleware.AuthRanger, userTicketHandler.ShowByPublicID)
	admin_api.GET("/user_tickets", authMiddleware.AuthRanger, userTicketHandler.Index)

	r.GET("/ws/events/:id", wsHandler.ConnectWS)

	/* Polls */
	admin_api.POST("/polls", authMiddleware.AuthAdmin, pollHandler.Create)
	admin_api.GET("/polls/event/:event_id", authMiddleware.AuthAdmin, pollHandler.IndexByEventID)
	admin_api.PUT("/polls/:id", authMiddleware.AuthAdmin, pollHandler.Update)
	admin_api.PUT("/polls/:id/status", authMiddleware.AuthAdmin, pollHandler.UpdateStatus)
	admin_api.DELETE("/polls/:id", authMiddleware.AuthAdmin, pollHandler.Delete)
	api.GET("/polls/:id", pollHandler.Show)
	api.GET("/polls/event/:event_id/active", pollHandler.ActiveByEventID)
	// Tamu tetap boleh menjawab poll, tetapi jawabannya tidak ditandai
	// terverifikasi dan tidak dihitung sebagai aktivitas akun.
	api.POST("/polls/:id/respond", authMiddleware.AuthOptionalUser, pollHandler.SubmitResponse)
	api.GET("/polls/:id/results", pollHandler.GetResults)

	/* Fase 1 — profil komunitas (F1-01) */
	// Identitas publik saja yang dibuka; profil privat/diblokir menjawab 404.
	api.GET("/community/profiles/:public_id", authMiddleware.AuthOptionalUser, communityHandler.ShowPublic)
	user_api.GET("/community/profile", authMiddleware.AuthUser, communityHandler.Me)
	user_api.PUT("/community/profile", authMiddleware.AuthUser, communityHandler.Update)

	/* Fase 1 — XP dan level (F1-02, F1-03) */
	api.GET("/leaderboard", authMiddleware.AuthOptionalUser, gamificationHandler.Leaderboard)
	user_api.GET("/xp", authMiddleware.AuthUser, gamificationHandler.Summary)
	user_api.GET("/xp/history", authMiddleware.AuthUser, gamificationHandler.History)
	admin_api.POST("/xp/adjustments", authMiddleware.AuthPJ, gamificationHandler.Adjust)
	admin_api.GET("/xp/rules", authMiddleware.AuthPJ, gamificationHandler.XPRules)
	admin_api.PUT("/xp/rules/:source_type", authMiddleware.AuthPJ, gamificationHandler.UpdateXPRule)
	admin_api.GET("/level_rules", authMiddleware.AuthPJ, gamificationHandler.LevelRules)
	admin_api.PUT("/level_rules", authMiddleware.AuthPJ, gamificationHandler.UpdateLevelRules)
	// Riwayat koreksi XP dan perubahan aturannya. Modul dana, moderasi, dan
	// toko punya endpoint riwayatnya sendiri; yang ini hanya memuat baris
	// bertipe entitas milik gamification.
	admin_api.GET("/xp/audit_logs", authMiddleware.AuthPJ, gamificationHandler.AuditLogs)

	/* Fase 1 — misi (F1-04, F1-05, F1-06) */
	api.GET("/missions", authMiddleware.AuthOptionalUser, missionHandler.ListPublished)
	api.GET("/missions/:id", authMiddleware.AuthOptionalUser, missionHandler.ShowPublished)
	user_api.POST("/missions/:id/claims", authMiddleware.AuthUser, missionHandler.Claim)
	user_api.GET("/missions/claims", authMiddleware.AuthUser, missionHandler.MyClaims)
	admin_api.GET("/missions", authMiddleware.AuthPJ, missionHandler.ListAll)
	admin_api.POST("/missions", authMiddleware.AuthPJ, missionHandler.Create)
	admin_api.PUT("/missions/:id", authMiddleware.AuthPJ, missionHandler.Update)
	admin_api.DELETE("/missions/:id", authMiddleware.AuthPJ, missionHandler.Delete)
	admin_api.GET("/missions/claims", authMiddleware.AuthPJ, missionHandler.ClaimsForReview)
	admin_api.PUT("/missions/claims/:id/approve", authMiddleware.AuthPJ, missionHandler.Approve)
	admin_api.PUT("/missions/claims/:id/reject", authMiddleware.AuthPJ, missionHandler.Reject)

	/* Fase 2 — fundraising (F2-01..F2-07) */
	// Permukaan publik. Parameter segmen bernama :slug, bukan :id, supaya
	// rujukan publik selalu lewat slug dan tidak pernah menebak id internal.
	api.GET("/campaigns", authMiddleware.AuthOptionalUser, fundraisingHandler.ListCampaigns)
	api.GET("/campaigns/:slug", authMiddleware.AuthOptionalUser, fundraisingHandler.ShowCampaign)
	api.GET("/campaigns/:slug/donors", authMiddleware.AuthOptionalUser, fundraisingHandler.ListDonors)
	api.GET("/campaigns/:slug/messages", authMiddleware.AuthOptionalUser, fundraisingHandler.ListMessages)
	api.GET("/campaigns/:slug/updates", authMiddleware.AuthOptionalUser, fundraisingHandler.ListUpdates)
	api.GET("/campaigns/:slug/report", authMiddleware.AuthOptionalUser, fundraisingHandler.PublicReport)

	// Anggota. Identitas donatur diambil dari konteks auth, bukan dari body.
	user_api.POST("/donations", authMiddleware.AuthUser, fundraisingHandler.CreateDonation)
	user_api.GET("/donations", authMiddleware.AuthUser, fundraisingHandler.MyDonations)
	user_api.GET("/donations/:public_id", authMiddleware.AuthUser, fundraisingHandler.MyDonation)

	// Pengurus. Izin funds:manage diperiksa di dalam handler.
	admin_api.GET("/campaigns", authMiddleware.AuthPJ, fundraisingHandler.ListAllCampaigns)
	admin_api.POST("/campaigns", authMiddleware.AuthPJ, fundraisingHandler.CreateCampaign)
	admin_api.PUT("/campaigns/:id", authMiddleware.AuthPJ, fundraisingHandler.UpdateCampaign)
	admin_api.DELETE("/campaigns/:id", authMiddleware.AuthPJ, fundraisingHandler.DeleteCampaign)
	admin_api.PUT("/campaigns/:id/status", authMiddleware.AuthPJ, fundraisingHandler.SetCampaignStatus)
	admin_api.GET("/campaigns/:id/report", authMiddleware.AuthPJ, fundraisingHandler.CampaignReport)
	admin_api.POST("/campaigns/:id/updates", authMiddleware.AuthPJ, fundraisingHandler.CreateUpdate)
	admin_api.PUT("/campaign_updates/:id", authMiddleware.AuthPJ, fundraisingHandler.UpdateUpdate)
	admin_api.DELETE("/campaign_updates/:id", authMiddleware.AuthPJ, fundraisingHandler.DeleteUpdate)
	admin_api.GET("/donations", authMiddleware.AuthPJ, fundraisingHandler.DonationsForReview)
	admin_api.PUT("/donations/:id/confirm", authMiddleware.AuthPJ, fundraisingHandler.ConfirmDonation)
	admin_api.PUT("/donations/:id/reject", authMiddleware.AuthPJ, fundraisingHandler.RejectDonation)
	admin_api.PUT("/donations/:id/refund", authMiddleware.AuthPJ, fundraisingHandler.RefundDonation)
	admin_api.PUT("/donations/:id/message", authMiddleware.AuthPJ, fundraisingHandler.ModerateMessage)
	admin_api.GET("/audit_logs", authMiddleware.AuthPJ, fundraisingHandler.ListAuditLogs)

	/* Fase 3 — threads anonim (F3-01..F3-04) */
	// Permukaan publik. Baca terbuka untuk tamu; identitas penulis selalu
	// alias, dan akun tanpa alias tampil sebagai "Anonim".
	api.GET("/threads", authMiddleware.AuthOptionalUser, threadHandler.ListThreads)
	api.GET("/threads/:public_id", authMiddleware.AuthOptionalUser, threadHandler.ShowThread)
	api.GET("/threads/:public_id/comments", authMiddleware.AuthOptionalUser, threadHandler.ListComments)

	// Anggota. Identitas penulis diambil dari konteks auth, bukan dari body.
	// Pembatas laju diperiksa di dalam service, bukan di sini.
	user_api.POST("/threads", authMiddleware.AuthUser, threadHandler.CreateThread)
	user_api.DELETE("/threads/:public_id", authMiddleware.AuthUser, threadHandler.DeleteThread)
	user_api.POST("/threads/:public_id/comments", authMiddleware.AuthUser, threadHandler.CreateComment)
	user_api.DELETE("/threads/:public_id/comments/:comment_public_id", authMiddleware.AuthUser, threadHandler.DeleteComment)
	user_api.PUT("/threads/:public_id/reaction", authMiddleware.AuthUser, threadHandler.React)
	user_api.DELETE("/threads/:public_id/reaction", authMiddleware.AuthUser, threadHandler.Unreact)
	user_api.POST("/threads/reports", authMiddleware.AuthUser, threadHandler.Report)

	// Preferensi berbagi aktivitas. Rutenya di bawah prefiks community karena
	// ia satu halaman pengaturan dengan profil komunitas; handler-nya tetap
	// milik paket thread, karena hanya dia yang tahu urusan auto-post.
	user_api.GET("/community/share_prefs", authMiddleware.AuthUser, threadHandler.MySharePrefs)
	user_api.PUT("/community/share_prefs", authMiddleware.AuthUser, threadHandler.UpdateSharePrefs)

	// Moderasi. AuthRanger meloloskan ranger, pj, dan admin; izin
	// moderation:moderate diperiksa lagi di dalam handler. AuthPJ tidak bisa
	// dipakai di sini karena ia menolak ranger, padahal ranger memegang izin itu.
	//
	// Riwayatnya memakai /moderation/audit_logs, bukan /audit_logs: yang
	// terakhir sudah dipakai jejak dana sejak Fase 2.
	admin_api.GET("/moderation/reports", authMiddleware.AuthRanger, threadHandler.ListReports)
	admin_api.PUT("/moderation/reports/:id/action", authMiddleware.AuthRanger, threadHandler.ActionReport)
	admin_api.PUT("/moderation/reports/:id/dismiss", authMiddleware.AuthRanger, threadHandler.DismissReport)
	admin_api.GET("/moderation/threads", authMiddleware.AuthRanger, threadHandler.ListThreadsForReview)
	admin_api.PUT("/moderation/threads/:id/:action", authMiddleware.AuthRanger, threadHandler.ModerateThread)
	admin_api.PUT("/moderation/thread_comments/:id/:action", authMiddleware.AuthRanger, threadHandler.ModerateComment)
	admin_api.GET("/moderation/accounts/restricted", authMiddleware.AuthRanger, threadHandler.RestrictedAccounts)
	admin_api.PUT("/moderation/accounts/:public_id/restrict", authMiddleware.AuthRanger, threadHandler.RestrictAccount)
	admin_api.GET("/moderation/audit_logs", authMiddleware.AuthRanger, threadHandler.ModerationAudit)

	/* Fase 4 — marketplace merchandise (F4-01..F4-06) */
	// Katalog terbuka untuk tamu: melihat barang tidak menuntut identitas.
	api.GET("/shop/products", authMiddleware.AuthOptionalUser, shopHandler.ListProducts)
	api.GET("/shop/products/:slug", authMiddleware.AuthOptionalUser, shopHandler.ShowProduct)

	// Pesanan anggota. Kepemilikannya diperiksa di service: pesanan milik orang
	// lain dijawab "tidak ditemukan".
	user_api.POST("/shop/orders", authMiddleware.AuthUser, shopHandler.Checkout)
	user_api.GET("/shop/orders", authMiddleware.AuthUser, shopHandler.MyOrders)
	user_api.GET("/shop/orders/:public_id", authMiddleware.AuthUser, shopHandler.MyOrder)
	user_api.PUT("/shop/orders/:public_id/proof", authMiddleware.AuthUser, shopHandler.SubmitProof)
	user_api.PUT("/shop/orders/:public_id/cancel", authMiddleware.AuthUser, shopHandler.CancelOrder)

	// Pengurus. AuthPJ meloloskan admin dan pj — himpunan yang sama persis
	// dengan pemegang product:manage; izinnya tetap diperiksa lagi di handler.
	//
	// Riwayatnya memakai /shop/audit_logs, bukan /audit_logs: yang terakhir
	// sudah dipakai jejak dana sejak Fase 2, dan /moderation/audit_logs dipakai
	// moderasi sejak Fase 3.
	admin_api.GET("/shop/products", authMiddleware.AuthPJ, shopHandler.AdminProducts)
	admin_api.POST("/shop/products", authMiddleware.AuthPJ, shopHandler.CreateProduct)
	admin_api.PUT("/shop/products/:id", authMiddleware.AuthPJ, shopHandler.UpdateProduct)
	admin_api.PUT("/shop/products/:id/status", authMiddleware.AuthPJ, shopHandler.SetProductStatus)
	admin_api.DELETE("/shop/products/:id", authMiddleware.AuthPJ, shopHandler.DeleteProduct)
	admin_api.POST("/shop/products/:id/variants", authMiddleware.AuthPJ, shopHandler.CreateVariant)
	admin_api.PUT("/shop/variants/:id", authMiddleware.AuthPJ, shopHandler.UpdateVariant)
	admin_api.DELETE("/shop/variants/:id", authMiddleware.AuthPJ, shopHandler.DeleteVariant)
	admin_api.POST("/shop/variants/:id/stock", authMiddleware.AuthPJ, shopHandler.AdjustStock)

	admin_api.GET("/shop/orders", authMiddleware.AuthPJ, shopHandler.AdminOrders)
	admin_api.PUT("/shop/orders/:id/confirm", authMiddleware.AuthPJ, shopHandler.ConfirmOrder)
	admin_api.PUT("/shop/orders/:id/reject", authMiddleware.AuthPJ, shopHandler.RejectOrder)
	admin_api.PUT("/shop/orders/:id/refund", authMiddleware.AuthPJ, shopHandler.RefundOrder)
	admin_api.PUT("/shop/orders/:id/fulfill", authMiddleware.AuthPJ, shopHandler.FulfillOrder)
	admin_api.PUT("/shop/orders/:id/shipping", authMiddleware.AuthPJ, shopHandler.SetShipping)
	admin_api.GET("/shop/audit_logs", authMiddleware.AuthPJ, shopHandler.AuditLogs)

	/* Lintas modul — laporan agregat */
	// Satu endpoint baca, tanpa penulisan dan tanpa tabel baru. Isinya hanya
	// angka: tidak ada id akun, nama, maupun email. Izinnya metrics:view,
	// dipegang admin dan pj.
	admin_api.GET("/metrics", authMiddleware.AuthPJ, metricsHandler.Show)
}

func Start(addr string) error {
	log.Printf("Server runing on %s", addr)
	return r.Run(addr)
}
