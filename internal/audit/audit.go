// Package audit menyediakan satu-satunya penulis untuk tabel audit_logs.
//
// Tabel ini lintas modul sejak awal: koreksi XP, keputusan dana, dan tindakan
// moderasi semuanya berjejak di sini. Sebelumnya penulisnya privat milik
// internal/fundraising; paket ini mengangkat bagian yang benar-benar bersama
// supaya modul berikutnya tidak perlu mengimpor fundraising hanya untuk
// mencatat satu keputusan.
//
// Yang TIDAK ikut diangkat: disiplin pemanggilnya. Audit hanya boleh ditulis
// bila penulisan terjaga yang mendahuluinya benar-benar mengubah baris
// (RowsAffected > 0), dan kegagalan menulis audit dilaporkan ke log, bukan
// dikembalikan sebagai galat. Itu tetap tanggung jawab pemanggil, karena hanya
// pemanggil yang tahu apa yang sudah terjadi.
package audit

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Log adalah satu baris jejak audit. Append-only: tidak ada deleted_at dan
// tidak ada jalur update, sama seperti xp_ledger.
type Log struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	ActorUserID *string   `json:"actor_user_id" gorm:"column:actor_user_id"`
	Action      string    `json:"action" gorm:"column:action"`
	EntityType  string    `json:"entity_type" gorm:"column:entity_type"`
	EntityID    string    `json:"entity_id" gorm:"column:entity_id"`
	Reason      *string   `json:"reason" gorm:"column:reason"`
	Detail      *string   `json:"detail" gorm:"column:detail"`
	DedupKey    string    `json:"-" gorm:"column:dedup_key"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

func (Log) TableName() string {
	return "audit_logs"
}

// Recorder menulis satu baris audit. Implementasi produksinya adalah
// NewRecorder; interface ini ada supaya pemanggil dapat menyuntikkan
// pencatat palsu pada test tanpa database.
type Recorder interface {
	Write(ctx *gin.Context, entry *Log) error
}

type recorder struct {
	db *gorm.DB
}

// NewRecorder mengembalikan penulis audit yang terhubung ke database.
func NewRecorder(db *gorm.DB) Recorder {
	return &recorder{db: db}
}

// Write menulis satu baris audit.
//
// dedup_key diisi ID entri oleh NewEntry, sehingga ON CONFLICT DO NOTHING di
// sini hanya berperan sebagai jaring pengaman untuk pemanggilan yang benar-benar
// identik — bukan sebagai mekanisme idempotensi utama. Idempotensi sesungguhnya
// datang dari penjagaan baris di pemanggil.
//
// detail dikirim sebagai teks JSON dan di-cast ::jsonb di sini, karena kolomnya
// bertipe jsonb.
func (r *recorder) Write(c *gin.Context, entry *Log) error {
	return r.db.Exec(
		`INSERT INTO audit_logs
		   (id, actor_user_id, action, entity_type, entity_id, reason, detail, dedup_key, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?)
		 ON CONFLICT (dedup_key) DO NOTHING`,
		entry.ID, entry.ActorUserID, entry.Action, entry.EntityType, entry.EntityID,
		entry.Reason, entry.Detail, entry.DedupKey, entry.CreatedAt,
	).Error
}

// NewEntry menyusun baris audit yang siap dikirim ke Write.
//
// actorID boleh kosong bila pelakunya tidak diketahui atau bukan pengurus;
// dalam hal itu actor_user_id dibiarkan NULL. detail boleh nil. Kegagalan
// mengubah detail menjadi JSON ditelan dengan sengaja: jejak audit tanpa detail
// masih jauh lebih berguna daripada tidak ada jejak sama sekali.
func NewEntry(actorID, action, entityType, entityID string, reason *string, detail map[string]any) *Log {
	entry := &Log{
		ID:         uuid.NewString(),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}
	if actorID != "" {
		actor := actorID
		entry.ActorUserID = &actor
	}
	if len(detail) > 0 {
		if encoded, err := json.Marshal(detail); err == nil {
			text := string(encoded)
			entry.Detail = &text
		}
	}
	// dedup_key = id entri: setiap keputusan punya identitasnya sendiri.
	// Idempotensi datang dari penjagaan baris di pemanggil, bukan dari kunci
	// ini — lihat komentar paket.
	entry.DedupKey = entry.ID
	return entry
}
