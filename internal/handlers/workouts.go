package handlers

import (
	"net/http"
	"strconv"
	"workout-tracker/internal/database"
	"workout-tracker/internal/models"

	"github.com/gin-gonic/gin"
)

type WorkoutHandler struct {
	db *database.DB
}

func NewWorkoutHandler(db *database.DB) *WorkoutHandler {
	return &WorkoutHandler{db: db}
}

// POST /workouts
func (h *WorkoutHandler) Create(c *gin.Context) {
	userID := c.GetInt64("userID")
	var req models.CreateWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	workout, err := h.db.CreateWorkout(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, workout)
}

// GET /workouts
func (h *WorkoutHandler) List(c *gin.Context) {
	userID := c.GetInt64("userID")
	status := c.Query("status") // pending, active, completed, or empty for all
	workouts, err := h.db.ListWorkouts(userID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if workouts == nil {
		workouts = []models.Workout{}
	}
	c.JSON(http.StatusOK, workouts)
}

// GET /workouts/:id
func (h *WorkoutHandler) Get(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	workout, err := h.db.GetWorkoutByID(id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if workout == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		return
	}
	c.JSON(http.StatusOK, workout)
}

// PUT /workouts/:id
func (h *WorkoutHandler) Update(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.UpdateWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	workout, err := h.db.UpdateWorkout(id, userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, workout)
}

// DELETE /workouts/:id
func (h *WorkoutHandler) Delete(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.db.DeleteWorkout(id, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GET /workouts/report
func (h *WorkoutHandler) Report(c *gin.Context) {
	userID := c.GetInt64("userID")
	report, err := h.db.GetReport(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
