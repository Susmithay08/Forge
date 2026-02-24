package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"workout-tracker/internal/database"
	"workout-tracker/internal/handlers"
	"workout-tracker/internal/middleware"
	"workout-tracker/internal/seeder"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *database.DB) {
	gin.SetMode(gin.TestMode)

	db, err := database.New(":memory:")
	if err != nil {
		t.Fatal("db open:", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatal("migrate:", err)
	}
	seeder.SeedExercises(db)

	r := gin.New()
	authH := handlers.NewAuthHandler(db)
	exH := handlers.NewExerciseHandler(db)
	workH := handlers.NewWorkoutHandler(db)

	r.POST("/auth/register", authH.Register)
	r.POST("/auth/login", authH.Login)
	r.GET("/auth/me", middleware.AuthRequired(), authH.Me)
	r.GET("/exercises", exH.List)

	protected := r.Group("/workouts", middleware.AuthRequired())
	protected.POST("", workH.Create)
	protected.GET("", workH.List)
	protected.GET("/report", workH.Report)
	protected.GET("/:id", workH.Get)
	protected.PUT("/:id", workH.Update)
	protected.DELETE("/:id", workH.Delete)

	return r, db
}

func TestRegisterAndLogin(t *testing.T) {
	r, _ := setupTestRouter(t)

	body, _ := json.Marshal(map[string]string{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "secret123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Register expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil {
		t.Fatal("Expected token in response")
	}

	// Login
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Login expected 200, got %d", w2.Code)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	r, _ := setupTestRouter(t)
	body, _ := json.Marshal(map[string]string{"name": "A", "email": "dup@test.com", "password": "pass123"})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if i == 1 && w.Code != http.StatusConflict {
			t.Fatalf("Expected 409 on duplicate, got %d", w.Code)
		}
	}
}

func registerAndGetToken(t *testing.T, r *gin.Engine, email string) string {
	body, _ := json.Marshal(map[string]string{"name": "User", "email": email, "password": "pass123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["token"].(string)
}

func TestCreateAndListWorkouts(t *testing.T) {
	r, _ := setupTestRouter(t)
	token := registerAndGetToken(t, r, "workout@test.com")

	body, _ := json.Marshal(map[string]interface{}{
		"title":       "Chest Day",
		"description": "Monday chest workout",
		"exercises": []map[string]interface{}{
			{"exercise_id": 1, "sets": 3, "reps": 10, "weight_kg": 60},
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/workouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create workout expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/workouts", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("List workouts expected 200, got %d", w2.Code)
	}

	var workouts []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &workouts)
	if len(workouts) != 1 {
		t.Fatalf("Expected 1 workout, got %d", len(workouts))
	}
}

func TestUpdateAndDeleteWorkout(t *testing.T) {
	r, _ := setupTestRouter(t)
	token := registerAndGetToken(t, r, "update@test.com")

	// Create
	body, _ := json.Marshal(map[string]interface{}{"title": "Leg Day"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/workouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	var workout map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &workout)
	id := int(workout["id"].(float64))

	// Update
	updateBody, _ := json.Marshal(map[string]interface{}{
		"status":  "completed",
		"comment": "Great session!",
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("PUT", "/workouts/"+itoa(id), bytes.NewBuffer(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Update expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Delete
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("DELETE", "/workouts/"+itoa(id), nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("Delete expected 200, got %d", w3.Code)
	}
}

func TestExercisesList(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/exercises", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/workouts", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", w.Code)
	}
}

func TestReport(t *testing.T) {
	r, _ := setupTestRouter(t)
	token := registerAndGetToken(t, r, "report@test.com")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/workouts/report", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func itoa(i int) string {
	return string(rune('0' + i)) // works for single digit IDs in tests
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
