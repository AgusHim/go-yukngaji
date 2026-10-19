package ticket

import (
	"errors"
	"strings"
	"time"

	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Episode adalah satu pertemuan dalam rangkaian event. Content sengaja
// dipisahkan agar materi dan rekaman dapat terbit pada waktu berbeda.
type Episode struct {
	ID          string     `json:"id"`
	EventID     string     `json:"event_id"`
	Position    int        `json:"position"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	IsPublished bool       `json:"is_published"`
	Contents    []Content  `json:"contents,omitempty" gorm:"-"`
	Unlocked    bool       `json:"unlocked" gorm:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `json:"-"`
}

func (Episode) TableName() string { return "event_episodes" }

type Content struct {
	ID          string     `json:"id"`
	EpisodeID   string     `json:"episode_id"`
	Position    int        `json:"position"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	URL         string     `json:"url"`
	IsPublished bool       `json:"is_published"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `json:"-"`
}

func (Content) TableName() string { return "event_episode_contents" }

type CreateEpisode struct {
	EventID     string  `json:"event_id" binding:"required"`
	Position    int     `json:"position" binding:"required"`
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	StartAt     *string `json:"start_at"`
	EndAt       *string `json:"end_at"`
	IsPublished bool    `json:"is_published"`
}

type CreateContent struct {
	EpisodeID   string `json:"episode_id" binding:"required"`
	Position    int    `json:"position" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	URL         string `json:"url" binding:"required"`
	IsPublished bool   `json:"is_published"`
}

func parseOptionalTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02T15:04", strings.TrimSuffix(*value, "Z"))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s *service) CreateEpisode(c *gin.Context, req *CreateEpisode) (*Episode, error) {
	startAt, err := parseOptionalTime(req.StartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := parseOptionalTime(req.EndAt)
	if err != nil {
		return nil, err
	}
	episode := &Episode{ID: uuid.NewString(), EventID: req.EventID, Position: req.Position, Title: req.Title, Description: req.Description, StartAt: startAt, EndAt: endAt, IsPublished: req.IsPublished, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.repositoryDB().Create(episode).Error; err != nil {
		return nil, err
	}
	return episode, nil
}

func (s *service) UpdateEpisode(c *gin.Context, id string, req *CreateEpisode) (*Episode, error) {
	startAt, err := parseOptionalTime(req.StartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := parseOptionalTime(req.EndAt)
	if err != nil {
		return nil, err
	}
	result := s.repositoryDB().Model(&Episode{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]interface{}{"event_id": req.EventID, "position": req.Position, "title": req.Title, "description": req.Description, "start_at": startAt, "end_at": endAt, "is_published": req.IsPublished, "updated_at": time.Now()})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.episodeByID(id)
}

func (s *service) DeleteEpisode(c *gin.Context, id string) error {
	now := time.Now()
	return s.repositoryDB().Model(&Episode{}).Where("id = ?", id).Updates(map[string]interface{}{"deleted_at": now, "updated_at": now}).Error
}

func (s *service) CreateContent(c *gin.Context, req *CreateContent) (*Content, error) {
	if req.Type != "material" && req.Type != "recording" {
		return nil, errors.New("type konten harus material atau recording")
	}
	content := &Content{ID: uuid.NewString(), EpisodeID: req.EpisodeID, Position: req.Position, Type: req.Type, Title: req.Title, Description: req.Description, URL: req.URL, IsPublished: req.IsPublished, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.repositoryDB().Create(content).Error; err != nil {
		return nil, err
	}
	return content, nil
}

func (s *service) UpdateContent(c *gin.Context, id string, req *CreateContent) (*Content, error) {
	if req.Type != "material" && req.Type != "recording" {
		return nil, errors.New("type konten harus material atau recording")
	}
	result := s.repositoryDB().Model(&Content{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]interface{}{"episode_id": req.EpisodeID, "position": req.Position, "type": req.Type, "title": req.Title, "description": req.Description, "url": req.URL, "is_published": req.IsPublished, "updated_at": time.Now()})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	content := &Content{}
	return content, s.repositoryDB().Where("id = ?", id).First(content).Error
}

func (s *service) DeleteContent(c *gin.Context, id string) error {
	now := time.Now()
	return s.repositoryDB().Model(&Content{}).Where("id = ?", id).Updates(map[string]interface{}{"deleted_at": now, "updated_at": now}).Error
}

func (s *service) episodeByID(id string) (*Episode, error) {
	episode := &Episode{}
	return episode, s.repositoryDB().Where("id = ? AND deleted_at IS NULL", id).First(episode).Error
}

func (s *service) ListEpisodes(c *gin.Context, eventID string, includeUnpublished bool) ([]*Episode, error) {
	var episodes []*Episode
	query := s.repositoryDB().Where("event_id = ? AND deleted_at IS NULL", eventID)
	if !includeUnpublished {
		query = query.Where("is_published = ?", true)
	}
	err := query.Order("position ASC").Find(&episodes).Error
	return episodes, err
}

// Learning mengembalikan metadata untuk semua episode, tetapi URL hanya pada
// episode yang dibuka tiket paid milik pengguna. Karena URL tidak pernah
// dikirim untuk episode terkunci, tautan rekaman tidak dapat diperoleh lewat
// API ini tanpa hak akses.
func (s *service) Learning(c *gin.Context, slug string) ([]*Episode, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, errors.New("unauthorized")
	}
	var eventRow struct{ ID string }
	if err := s.repositoryDB().Table("events").Select("id").Where("slug = ? AND deleted_at IS NULL", slug).First(&eventRow).Error; err != nil {
		return nil, err
	}
	episodes, err := s.ListEpisodes(c, eventRow.ID, false)
	if err != nil {
		return nil, err
	}
	if len(episodes) == 0 {
		return episodes, nil
	}

	paidTickets := func() *gorm.DB {
		return s.repositoryDB().Table("user_tickets ut").Joins("JOIN orders o ON o.id = ut.order_id").Joins("JOIN tickets t ON t.id = ut.ticket_id").Where("ut.event_id = ? AND o.status = ? AND ut.deleted_at IS NULL", eventRow.ID, "paid").Where("(ut.user_id = ? OR ut.participant_user_id = ?)", currentUser.ID, currentUser.ID)
	}
	var allAccess int64
	if err := paidTickets().Where("t.access_mode = ?", "all").Count(&allAccess).Error; err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	if allAccess > 0 {
		for _, episode := range episodes {
			allowed[episode.ID] = true
		}
	} else {
		var episodeIDs []string
		err = paidTickets().Joins("JOIN ticket_episode_access tea ON tea.ticket_id = t.id").Pluck("tea.episode_id", &episodeIDs).Error
		if err != nil {
			return nil, err
		}
		for _, id := range episodeIDs {
			allowed[id] = true
		}
	}

	for _, episode := range episodes {
		episode.Unlocked = allowed[episode.ID]
		if !episode.Unlocked {
			continue
		}
		if err := s.repositoryDB().Where("episode_id = ? AND is_published = ? AND deleted_at IS NULL", episode.ID, true).Order("position ASC").Find(&episode.Contents).Error; err != nil {
			return nil, err
		}
	}
	return episodes, nil
}

// repositoryDB menjaga detail GORM tetap berada di package ticket sambil
// mempertahankan antarmuka repository lama yang dipakai order.
func (s *service) repositoryDB() *gorm.DB { return s.Repository.(*repository).db }
