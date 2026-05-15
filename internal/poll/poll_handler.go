package poll

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return &handler{service: service}
}

func (h *handler) Create(c *gin.Context) {
	var req CreatePoll
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poll, err := h.service.Create(c, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, poll)
}

func (h *handler) IndexByEventID(c *gin.Context) {
	eventID := c.Param("event_id")
	polls, err := h.service.IndexByEventID(c, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, polls)
}

func (h *handler) Show(c *gin.Context) {
	id := c.Param("id")
	poll, err := h.service.Show(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}
	c.JSON(http.StatusOK, poll)
}

func (h *handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdatePoll
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poll, err := h.service.Update(c, id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, poll)
}

func (h *handler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdatePollStatus
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.UpdateStatus(c, id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

func (h *handler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.service.Delete(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Poll deleted"})
}

func (h *handler) SubmitResponse(c *gin.Context) {
	pollID := c.Param("id")
	var req SubmitResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.SubmitResponse(c, pollID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, response)
}

func (h *handler) GetResults(c *gin.Context) {
	pollID := c.Param("id")
	results, err := h.service.GetResults(c, pollID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

func (h *handler) ActiveByEventID(c *gin.Context) {
	eventID := c.Param("event_id")
	poll, err := h.service.ActiveByEventID(c, eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active poll found"})
		return
	}
	c.JSON(http.StatusOK, poll)
}
