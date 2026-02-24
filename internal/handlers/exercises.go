package handlers

import (
	"net/http"
	"workout-tracker/internal/database"

	"github.com/gin-gonic/gin"
)

type ExerciseHandler struct {
	db *database.DB
}

func NewExerciseHandler(db *database.DB) *ExerciseHandler {
	return &ExerciseHandler{db: db}
}

// GET /exercises
func (h *ExerciseHandler) List(c *gin.Context) {
	exercises, err := h.db.GetExercises()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, exercises)
}
