package router

import (
	"strings"
	"testing"

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

	"github.com/gin-gonic/gin"
)

// TestInitRouterTidakPanik mendaftarkan seluruh tabel route yang sebenarnya.
//
// Gin menolak sebagian kombinasi segmen statis dan segmen ber-parameter pada
// posisi yang sama, dan penolakannya berupa panic saat pendaftaran — bukan
// error kompilasi. Karena itu tabel route Fase 1 (/missions/:id berdampingan
// dengan /missions/claims, dan /xp/rules/:source_type) perlu diuji sungguhan.
//
// Handler dibangun lewat konstruktor dengan service nil: yang diuji di sini
// adalah pendaftaran route, bukan perilakunya, jadi service tidak pernah
// dipanggil.
func TestInitRouterTidakPanik(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("pendaftaran route panik: %v", r)
		}
	}()

	authMiddleware := auth.NewMiddleware(nil)

	InitRouter(
		authMiddleware,
		user.NewHandler(nil),
		event.NewHandler(nil),
		divisi.NewHandler(nil),
		presence.NewHandler(nil),
		comment.NewHandler(nil),
		like.NewHandler(nil),
		feedback.NewHandler(nil),
		ws.NewHandler(ws.NewHub(), nil),
		agenda.NewHandler(nil),
		ranger.NewHandler(nil),
		ranger_presence.NewHandler(nil),
		order.NewHandler(nil),
		ticket.NewHandler(nil),
		user_ticket.NewHandler(nil),
		payment_method.NewHandler(nil),
		region.NewHandler(nil),
		otp.NewHandler(nil),
		poll.NewHandler(nil),
		community.NewHandler(nil),
		gamification.NewHandler(nil),
		mission.NewHandler(nil),
		fundraising.NewHandler(nil),
		thread.NewHandler(nil),
		shop.NewHandler(nil),
		metrics.NewHandler(nil),
	)

	if r == nil {
		t.Fatal("router tidak terbentuk")
	}
}

// TestRouteFase1Terdaftar memastikan setiap route Fase 1 benar-benar ada pada
// grup yang benar, sehingga perubahan tabel route tidak diam-diam menghapus
// salah satunya.
func TestRouteFase1Terdaftar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMiddleware := auth.NewMiddleware(nil)
	InitRouter(
		authMiddleware,
		user.NewHandler(nil),
		event.NewHandler(nil),
		divisi.NewHandler(nil),
		presence.NewHandler(nil),
		comment.NewHandler(nil),
		like.NewHandler(nil),
		feedback.NewHandler(nil),
		ws.NewHandler(ws.NewHub(), nil),
		agenda.NewHandler(nil),
		ranger.NewHandler(nil),
		ranger_presence.NewHandler(nil),
		order.NewHandler(nil),
		ticket.NewHandler(nil),
		user_ticket.NewHandler(nil),
		payment_method.NewHandler(nil),
		region.NewHandler(nil),
		otp.NewHandler(nil),
		poll.NewHandler(nil),
		community.NewHandler(nil),
		gamification.NewHandler(nil),
		mission.NewHandler(nil),
		fundraising.NewHandler(nil),
		thread.NewHandler(nil),
		shop.NewHandler(nil),
		metrics.NewHandler(nil),
	)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /api/leaderboard",
		"GET /api/community/profiles/:public_id",
		"GET /api/missions",
		"GET /api/missions/:id",

		"GET /user_api/community/profile",
		"PUT /user_api/community/profile",
		"GET /user_api/xp",
		"GET /user_api/xp/history",
		"POST /user_api/missions/:id/claims",
		"GET /user_api/missions/claims",

		"GET /admin_api/missions",
		"POST /admin_api/missions",
		"PUT /admin_api/missions/:id",
		"DELETE /admin_api/missions/:id",
		"GET /admin_api/missions/claims",
		"PUT /admin_api/missions/claims/:id/approve",
		"PUT /admin_api/missions/claims/:id/reject",
		"POST /admin_api/xp/adjustments",
		"GET /admin_api/xp/rules",
		"PUT /admin_api/xp/rules/:source_type",
		"GET /admin_api/level_rules",
		"PUT /admin_api/level_rules",
		"GET /admin_api/xp/audit_logs",
	}

	for _, route := range want {
		if !registered[route] {
			t.Errorf("route tidak terdaftar: %s", route)
		}
	}
}

// TestRouteFase2Terdaftar memastikan route fundraising terpasang pada grup
// yang benar. Yang paling rawan di sini adalah /campaigns/:slug (publik)
// berdampingan dengan /campaigns/:id/status dan /campaigns/:id/report
// (pengurus): Gin menolak dua nama parameter berbeda pada posisi segmen yang
// sama, dan penolakannya baru muncul saat pendaftaran.
func TestRouteFase2Terdaftar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMiddleware := auth.NewMiddleware(nil)
	InitRouter(
		authMiddleware,
		user.NewHandler(nil),
		event.NewHandler(nil),
		divisi.NewHandler(nil),
		presence.NewHandler(nil),
		comment.NewHandler(nil),
		like.NewHandler(nil),
		feedback.NewHandler(nil),
		ws.NewHandler(ws.NewHub(), nil),
		agenda.NewHandler(nil),
		ranger.NewHandler(nil),
		ranger_presence.NewHandler(nil),
		order.NewHandler(nil),
		ticket.NewHandler(nil),
		user_ticket.NewHandler(nil),
		payment_method.NewHandler(nil),
		region.NewHandler(nil),
		otp.NewHandler(nil),
		poll.NewHandler(nil),
		community.NewHandler(nil),
		gamification.NewHandler(nil),
		mission.NewHandler(nil),
		fundraising.NewHandler(nil),
		thread.NewHandler(nil),
		shop.NewHandler(nil),
		metrics.NewHandler(nil),
	)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /api/campaigns",
		"GET /api/campaigns/:slug",
		"GET /api/campaigns/:slug/donors",
		"GET /api/campaigns/:slug/messages",
		"GET /api/campaigns/:slug/updates",
		"GET /api/campaigns/:slug/report",

		"POST /user_api/donations",
		"GET /user_api/donations",
		"GET /user_api/donations/:public_id",

		"GET /admin_api/campaigns",
		"POST /admin_api/campaigns",
		"PUT /admin_api/campaigns/:id",
		"DELETE /admin_api/campaigns/:id",
		"PUT /admin_api/campaigns/:id/status",
		"GET /admin_api/campaigns/:id/report",
		"POST /admin_api/campaigns/:id/updates",
		"PUT /admin_api/campaign_updates/:id",
		"DELETE /admin_api/campaign_updates/:id",
		"GET /admin_api/donations",
		"PUT /admin_api/donations/:id/confirm",
		"PUT /admin_api/donations/:id/reject",
		"PUT /admin_api/donations/:id/refund",
		"PUT /admin_api/donations/:id/message",
		"GET /admin_api/audit_logs",
	}

	for _, route := range want {
		if !registered[route] {
			t.Errorf("route tidak terdaftar: %s", route)
		}
	}
}

// TestRouteFase3Terdaftar memastikan route feed komunitas dan moderasi
// terpasang pada grup yang benar.
//
// Dua hal yang paling rawan di sini, dan keduanya hanya muncul sebagai panic
// saat pendaftaran: POST /threads/reports berdampingan dengan
// POST /threads/:public_id/comments (segmen statis dan segmen ber-parameter
// pada posisi yang sama), dan PUT /moderation/threads/:id/:action yang
// menyerap hide, restore, dan delete dalam satu pola.
func TestRouteFase3Terdaftar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMiddleware := auth.NewMiddleware(nil)
	InitRouter(
		authMiddleware,
		user.NewHandler(nil),
		event.NewHandler(nil),
		divisi.NewHandler(nil),
		presence.NewHandler(nil),
		comment.NewHandler(nil),
		like.NewHandler(nil),
		feedback.NewHandler(nil),
		ws.NewHandler(ws.NewHub(), nil),
		agenda.NewHandler(nil),
		ranger.NewHandler(nil),
		ranger_presence.NewHandler(nil),
		order.NewHandler(nil),
		ticket.NewHandler(nil),
		user_ticket.NewHandler(nil),
		payment_method.NewHandler(nil),
		region.NewHandler(nil),
		otp.NewHandler(nil),
		poll.NewHandler(nil),
		community.NewHandler(nil),
		gamification.NewHandler(nil),
		mission.NewHandler(nil),
		fundraising.NewHandler(nil),
		thread.NewHandler(nil),
		shop.NewHandler(nil),
		metrics.NewHandler(nil),
	)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /api/threads",
		"GET /api/threads/:public_id",
		"GET /api/threads/:public_id/comments",

		"POST /user_api/threads",
		"DELETE /user_api/threads/:public_id",
		"POST /user_api/threads/:public_id/comments",
		"DELETE /user_api/threads/:public_id/comments/:comment_public_id",
		"PUT /user_api/threads/:public_id/reaction",
		"DELETE /user_api/threads/:public_id/reaction",
		"POST /user_api/threads/reports",
		"GET /user_api/community/share_prefs",
		"PUT /user_api/community/share_prefs",

		"GET /admin_api/moderation/reports",
		"PUT /admin_api/moderation/reports/:id/action",
		"PUT /admin_api/moderation/reports/:id/dismiss",
		"GET /admin_api/moderation/threads",
		"PUT /admin_api/moderation/threads/:id/:action",
		"PUT /admin_api/moderation/thread_comments/:id/:action",
		"GET /admin_api/moderation/accounts/restricted",
		"PUT /admin_api/moderation/accounts/:public_id/restrict",
		"GET /admin_api/moderation/audit_logs",
	}

	for _, route := range want {
		if !registered[route] {
			t.Errorf("route tidak terdaftar: %s", route)
		}
	}

	// Riwayat audit moderasi sengaja terpisah dari riwayat audit dana: satu
	// pohon route tidak boleh punya dua endpoint dengan path yang sama.
	if !registered["GET /admin_api/audit_logs"] {
		t.Error("route audit dana hilang; seharusnya tetap terdaftar")
	}
}

// TestRouteFase4Terdaftar memastikan setiap route Fase 4 benar-benar ada pada
// grup yang benar.
//
// Dua hal yang khusus diuji di sini karena keduanya pernah menjadi sumber
// kebingungan: GET /api/shop/products (publik) dan GET /admin_api/shop/products
// (pengurus) hidup di grup yang berbeda prefiks sehingga tidak bertabrakan, dan
// riwayat audit toko memakai /shop/audit_logs karena /audit_logs sudah dipakai
// jejak dana sejak Fase 2.
func TestRouteFase4Terdaftar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMiddleware := auth.NewMiddleware(nil)
	InitRouter(
		authMiddleware,
		user.NewHandler(nil),
		event.NewHandler(nil),
		divisi.NewHandler(nil),
		presence.NewHandler(nil),
		comment.NewHandler(nil),
		like.NewHandler(nil),
		feedback.NewHandler(nil),
		ws.NewHandler(ws.NewHub(), nil),
		agenda.NewHandler(nil),
		ranger.NewHandler(nil),
		ranger_presence.NewHandler(nil),
		order.NewHandler(nil),
		ticket.NewHandler(nil),
		user_ticket.NewHandler(nil),
		payment_method.NewHandler(nil),
		region.NewHandler(nil),
		otp.NewHandler(nil),
		poll.NewHandler(nil),
		community.NewHandler(nil),
		gamification.NewHandler(nil),
		mission.NewHandler(nil),
		fundraising.NewHandler(nil),
		thread.NewHandler(nil),
		shop.NewHandler(nil),
		metrics.NewHandler(nil),
	)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /api/shop/products",
		"GET /api/shop/products/:slug",

		"POST /user_api/shop/orders",
		"GET /user_api/shop/orders",
		"GET /user_api/shop/orders/:public_id",
		"PUT /user_api/shop/orders/:public_id/proof",
		"PUT /user_api/shop/orders/:public_id/cancel",

		"GET /admin_api/shop/products",
		"POST /admin_api/shop/products",
		"PUT /admin_api/shop/products/:id",
		"PUT /admin_api/shop/products/:id/status",
		"DELETE /admin_api/shop/products/:id",
		"POST /admin_api/shop/products/:id/variants",
		"PUT /admin_api/shop/variants/:id",
		"DELETE /admin_api/shop/variants/:id",
		"POST /admin_api/shop/variants/:id/stock",

		"GET /admin_api/shop/orders",
		"PUT /admin_api/shop/orders/:id/confirm",
		"PUT /admin_api/shop/orders/:id/reject",
		"PUT /admin_api/shop/orders/:id/refund",
		"PUT /admin_api/shop/orders/:id/fulfill",
		"PUT /admin_api/shop/orders/:id/shipping",
		"GET /admin_api/shop/audit_logs",
	}

	for _, route := range want {
		if !registered[route] {
			t.Errorf("route tidak terdaftar: %s", route)
		}
	}

	// Katalog publik dan katalog pengurus berbagi path yang sama di grup yang
	// berbeda prefiks; keduanya harus tetap ada.
	if !registered["GET /api/shop/products"] || !registered["GET /admin_api/shop/products"] {
		t.Error("katalog publik dan katalog pengurus harus sama-sama terdaftar")
	}

	// Riwayat audit toko sengaja tidak memakai /admin_api/audit_logs.
	if !registered["GET /admin_api/audit_logs"] {
		t.Error("route audit dana hilang; seharusnya tetap terdaftar")
	}
}

// TestRouteLintasModulTerdaftar memastikan endpoint yang ditambahkan pada
// pekerjaan lintas modul terpasang pada grup yang benar.
//
// Ketiganya adalah endpoint baca untuk pengurus: riwayat audit koreksi XP,
// daftar akun, dan laporan agregat. Yang paling mudah salah adalah grupnya —
// menaruhnya di /api akan membukanya untuk tamu tanpa peringatan apa pun.
func TestRouteLintasModulTerdaftar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMiddleware := auth.NewMiddleware(nil)
	InitRouter(
		authMiddleware,
		user.NewHandler(nil),
		event.NewHandler(nil),
		divisi.NewHandler(nil),
		presence.NewHandler(nil),
		comment.NewHandler(nil),
		like.NewHandler(nil),
		feedback.NewHandler(nil),
		ws.NewHandler(ws.NewHub(), nil),
		agenda.NewHandler(nil),
		ranger.NewHandler(nil),
		ranger_presence.NewHandler(nil),
		order.NewHandler(nil),
		ticket.NewHandler(nil),
		user_ticket.NewHandler(nil),
		payment_method.NewHandler(nil),
		region.NewHandler(nil),
		otp.NewHandler(nil),
		poll.NewHandler(nil),
		community.NewHandler(nil),
		gamification.NewHandler(nil),
		mission.NewHandler(nil),
		fundraising.NewHandler(nil),
		thread.NewHandler(nil),
		shop.NewHandler(nil),
		metrics.NewHandler(nil),
	)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /admin_api/xp/audit_logs",
		"GET /admin_api/users",
		"GET /admin_api/metrics",
	}

	for _, route := range want {
		if !registered[route] {
			t.Errorf("route tidak terdaftar: %s", route)
		}
	}

	// Tidak satu pun boleh muncul di grup terbuka.
	for _, route := range want {
		terbuka := "GET /api" + strings.TrimPrefix(route, "GET /admin_api")
		if registered[terbuka] {
			t.Errorf("route pengurus bocor ke grup publik: %s", terbuka)
		}
	}
}
