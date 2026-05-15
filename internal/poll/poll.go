package poll

import (
	"time"

	"github.com/gin-gonic/gin"
)

// ============ MODELS ============

type Poll struct {
	ID                   string       `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EventID              string       `json:"event_id"`
	Title                string       `json:"title"`
	Type                 string       `json:"type"`
	Status               string       `json:"status" gorm:"default:draft"`
	AllowMultipleAnswers bool         `json:"allow_multiple_answers" gorm:"default:false"`
	ShowResults          bool         `json:"show_results" gorm:"default:true"`
	OrderIndex           int          `json:"order_index" gorm:"default:0"`
	Options              []PollOption `json:"options" gorm:"foreignKey:PollID"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"-"`
	DeletedAt            *time.Time   `json:"-"`
}

type PollOption struct {
	ID         string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PollID     string     `json:"poll_id"`
	Text       string     `json:"text"`
	IsCorrect  bool       `json:"is_correct" gorm:"default:false"`
	OrderIndex int        `json:"order_index" gorm:"default:0"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"-"`
	DeletedAt  *time.Time `json:"-"`
}

type PollResponse struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PollID       string    `json:"poll_id"`
	PollOptionID *string   `json:"poll_option_id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	TextResponse *string   `json:"text_response"`
	Rank         *int      `json:"rank"`
	CreatedAt    time.Time `json:"created_at"`
}

// ============ DTOs ============

type CreatePoll struct {
	EventID              string             `json:"event_id" binding:"required"`
	Title                string             `json:"title" binding:"required"`
	Type                 string             `json:"type" binding:"required"`
	AllowMultipleAnswers bool               `json:"allow_multiple_answers"`
	ShowResults          bool               `json:"show_results"`
	Options              []CreatePollOption `json:"options"`
}

type CreatePollOption struct {
	Text      string `json:"text" binding:"required"`
	IsCorrect bool   `json:"is_correct"`
}

type UpdatePoll struct {
	Title                string             `json:"title"`
	Type                 string             `json:"type"`
	AllowMultipleAnswers *bool              `json:"allow_multiple_answers"`
	ShowResults          *bool              `json:"show_results"`
	Options              []CreatePollOption `json:"options"`
}

type UpdatePollStatus struct {
	Status string `json:"status" binding:"required"`
}

type SubmitResponse struct {
	PollOptionID *string `json:"poll_option_id"`
	UserID       string  `json:"user_id" binding:"required"`
	Username     string  `json:"username"`
	TextResponse *string `json:"text_response"`
	Rank         *int    `json:"rank"`
}

// ============ RESULT TYPES ============

type OptionResult struct {
	OptionID  string `json:"option_id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
	Count     int    `json:"count"`
	TotalRank int    `json:"total_rank"`
}

type PollResults struct {
	PollID        string         `json:"poll_id"`
	Type          string         `json:"type"`
	TotalVotes    int            `json:"total_votes"`
	Options       []OptionResult `json:"options"`
	TextResponses []PollResponse `json:"text_responses"`
}

// ============ INTERFACES ============

type Repository interface {
	Create(ctx *gin.Context, poll *Poll) (*Poll, error)
	IndexByEventID(ctx *gin.Context, eventID string) ([]*Poll, error)
	Show(ctx *gin.Context, id string) (*Poll, error)
	Update(ctx *gin.Context, id string, poll *Poll) (*Poll, error)
	UpdateStatus(ctx *gin.Context, id string, status string) error
	Delete(ctx *gin.Context, id string) error
	SaveResponse(ctx *gin.Context, response *PollResponse) (*PollResponse, error)
	GetResponses(ctx *gin.Context, pollID string) ([]PollResponse, error)
	DeleteOptionsByPollID(ctx *gin.Context, pollID string) error
	ActiveByEventID(ctx *gin.Context, eventID string) (*Poll, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreatePoll) (*Poll, error)
	IndexByEventID(ctx *gin.Context, eventID string) ([]*Poll, error)
	Show(ctx *gin.Context, id string) (*Poll, error)
	Update(ctx *gin.Context, id string, req *UpdatePoll) (*Poll, error)
	UpdateStatus(ctx *gin.Context, id string, req *UpdatePollStatus) error
	Delete(ctx *gin.Context, id string) error
	SubmitResponse(ctx *gin.Context, pollID string, req *SubmitResponse) (*PollResponse, error)
	GetResults(ctx *gin.Context, pollID string) (*PollResults, error)
	ActiveByEventID(ctx *gin.Context, eventID string) (*Poll, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	IndexByEventID(ctx *gin.Context)
	Show(ctx *gin.Context)
	Update(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	Delete(ctx *gin.Context)
	SubmitResponse(ctx *gin.Context)
	GetResults(ctx *gin.Context)
	ActiveByEventID(ctx *gin.Context)
}
