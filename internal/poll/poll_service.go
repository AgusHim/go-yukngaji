package poll

import (
	"errors"
	"strings"

	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(c *gin.Context, req *CreatePoll) (*Poll, error) {
	poll := &Poll{
		EventID:              req.EventID,
		Title:                req.Title,
		Type:                 req.Type,
		Status:               "draft",
		AllowMultipleAnswers: req.AllowMultipleAnswers,
		ShowResults:          req.ShowResults,
	}

	// Create options
	for i, opt := range req.Options {
		poll.Options = append(poll.Options, PollOption{
			Text:       opt.Text,
			IsCorrect:  opt.IsCorrect,
			OrderIndex: i,
		})
	}

	return s.repo.Create(c, poll)
}

func (s *service) IndexByEventID(c *gin.Context, eventID string) ([]*Poll, error) {
	return s.repo.IndexByEventID(c, eventID)
}

func (s *service) Show(c *gin.Context, id string) (*Poll, error) {
	return s.repo.Show(c, id)
}

func (s *service) Update(c *gin.Context, id string, req *UpdatePoll) (*Poll, error) {
	existing, err := s.repo.Show(c, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.AllowMultipleAnswers != nil {
		existing.AllowMultipleAnswers = *req.AllowMultipleAnswers
	}
	if req.ShowResults != nil {
		existing.ShowResults = *req.ShowResults
	}

	// If options provided, replace them
	if req.Options != nil {
		// Delete old options
		err = s.repo.DeleteOptionsByPollID(c, id)
		if err != nil {
			return nil, err
		}

		// Create new options
		existing.Options = nil
		for i, opt := range req.Options {
			existing.Options = append(existing.Options, PollOption{
				PollID:     id,
				Text:       opt.Text,
				IsCorrect:  opt.IsCorrect,
				OrderIndex: i,
			})
		}
	}

	return s.repo.Update(c, id, existing)
}

func (s *service) UpdateStatus(c *gin.Context, id string, req *UpdatePollStatus) error {
	validStatuses := map[string]bool{"draft": true, "active": true, "closed": true}
	if !validStatuses[req.Status] {
		return errors.New("invalid status: must be draft, active, or closed")
	}

	// If activating, close any other active polls for the same event
	if req.Status == "active" {
		poll, err := s.repo.Show(c, id)
		if err != nil {
			return err
		}
		activePoll, _ := s.repo.ActiveByEventID(c, poll.EventID)
		if activePoll != nil && activePoll.ID != id {
			_ = s.repo.UpdateStatus(c, activePoll.ID, "closed")
		}
	}

	return s.repo.UpdateStatus(c, id, req.Status)
}

func (s *service) Delete(c *gin.Context, id string) error {
	return s.repo.Delete(c, id)
}

// SubmitResponse menyimpan jawaban poll.
//
// Identitas diambil dari token, bukan dari body. Jawaban tamu tetap diterima
// tetapi disimpan tanpa user_id dan ditandai is_verified = false, sehingga
// tidak pernah dihitung sebagai aktivitas akun yang sah.
func (s *service) SubmitResponse(c *gin.Context, pollID string, req *SubmitResponse) (*PollResponse, error) {
	currentUser, loggedIn := user.FromContext(c)

	limitKey := currentUser.ID
	if !loggedIn {
		limitKey = ""
	}
	if !ratelimit.PollResponse.Allow(ratelimit.Key(c, limitKey)) {
		return nil, ratelimit.ErrTooManyRequests
	}

	poll, err := s.repo.Show(c, pollID)
	if err != nil {
		return nil, err
	}
	if poll.Status != "active" {
		return nil, errors.New("poll is not active")
	}

	response := &PollResponse{
		PollID:       pollID,
		PollOptionID: req.PollOptionID,
		TextResponse: req.TextResponse,
		Rank:         req.Rank,
		Username:     strings.TrimSpace(req.Username),
	}

	if loggedIn {
		userID := currentUser.ID
		response.UserID = &userID
		response.IsVerified = true
		if response.Username == "" {
			response.Username = currentUser.Username
		}
	} else if response.Username == "" {
		response.Username = "Anonim"
	}

	// Satu jawaban per akun untuk poll pilihan ganda tanpa multi-answer.
	// Tamu tidak bisa dideduplikasi tanpa mempercayai identitas dari client,
	// jadi pembatasnya adalah rate limit per IP di atas.
	if loggedIn && poll.Type != "word_cloud" && !poll.AllowMultipleAnswers {
		responses, errResponses := s.repo.GetResponses(c, pollID)
		if errResponses != nil {
			return nil, errResponses
		}
		for _, r := range responses {
			if r.UserID != nil && *r.UserID == currentUser.ID {
				return nil, errors.New("you have already responded to this poll")
			}
		}
	}

	return s.repo.SaveResponse(c, response)
}

func (s *service) GetResults(c *gin.Context, pollID string) (*PollResults, error) {
	poll, err := s.repo.Show(c, pollID)
	if err != nil {
		return nil, err
	}

	responses, err := s.repo.GetResponses(c, pollID)
	if err != nil {
		return nil, err
	}

	results := &PollResults{
		PollID:     pollID,
		Type:       poll.Type,
		TotalVotes: len(responses),
	}

	if poll.Type == "word_cloud" {
		results.TextResponses = responses
	} else {
		// Aggregate by option
		optionMap := make(map[string]*OptionResult)
		for _, opt := range poll.Options {
			optionMap[opt.ID] = &OptionResult{
				OptionID:  opt.ID,
				Text:      opt.Text,
				IsCorrect: opt.IsCorrect,
			}
		}

		for _, resp := range responses {
			if resp.PollOptionID != nil {
				if or, ok := optionMap[*resp.PollOptionID]; ok {
					or.Count++
					if resp.Rank != nil {
						or.TotalRank += *resp.Rank
					}
				}
			}
		}

		for _, opt := range poll.Options {
			if or, ok := optionMap[opt.ID]; ok {
				results.Options = append(results.Options, *or)
			}
		}
	}

	return results, nil
}

func (s *service) ActiveByEventID(c *gin.Context, eventID string) (*Poll, error) {
	return s.repo.ActiveByEventID(c, eventID)
}
