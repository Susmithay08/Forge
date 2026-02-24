package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "test-secret-key")

	db, _ := database.New(":memory:")
	db.Migrate()
	seeder.SeedExercises(db)

	r := gin.New()

	authH := handlers.NewAuthHandler(db)
	exerciseH := handlers.NewExerciseHandler(db)
	workoutH := handlers.NewWorkoutHandler(db)

	r.POST("/auth/register", authH.Register)
	r.POST("/auth/login", authH.Login)
	r.GET("/auth/me", middleware.AuthRequired(), authH.Me)

	r.GET("/exercises", middleware.AuthRequired(), exerciseH.List)

	wg := r.Group("/workouts", middleware.AuthRequired())
	{
		wg.POST("", workoutH.Create)
		wg.GET("", workoutH.List)
		wg.GET("/report", workoutH.Report)
		wg.GET("/:id", workoutH.Get)
		wg.PUT("/:id", workoutH.Update)
		wg.DELETE("/:id", workoutH.Delete)
	}

	return r
}

func performRequest(r *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func registerAndGetToken(r *gin.Engine, email string) string {
	body := map[string]string{"name": "Test User", "email": email, "password": "password123"}
	w := performRequest(r, "POST", "/auth/register", body, "")
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if t, ok := resp["token"].(string); ok {
		return t
	}
	return ""
}

func TestRegister_Success(t *testing.T) {
	r := setupTestRouter()
	body := map[string]string{"name": "Test User", "email": "test@example.com", "password": "password123"}
	w := performRequest(r, "POST", "/auth/register", body, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] == nil {
		t.Error("Expected token in response")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	r := setupTestRouter()
	body := map[string]string{"name": "Test", "email": "dup@example.com", "password": "password123"}
	performRequest(r, "POST", "/auth/register", body, "")
	w := performRequest(r, "POST", "/auth/register", body, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("Expected 409, got %d", w.Code)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	r := setupTestRouter()
	body := map[string]string{"name": "Test", "email": "not-an-email", "password": "password123"}
	w := performRequest(r, "POST", "/auth/register", body, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	r := setupTestRouter()
	performRequest(r, "POST", "/auth/register", map[string]string{"name": "Login User", "email": "login@example.com", "password": "securepass"}, "")
	w := performRequest(r, "POST", "/auth/login", map[string]string{"email": "login@example.com", "password": "securepass"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] == nil {
		t.Error("Expected token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	r := setupTestRouter()
	performRequest(r, "POST", "/auth/register", map[string]string{"name": "User", "email": "wrongpw@example.com", "password": "correctpass"}, "")
	w := performRequest(r, "POST", "/auth/login", map[string]string{"email": "wrongpw@example.com", "password": "wrongpass"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", w.Code)
	}
}

func TestMe_Authenticated(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "me@example.com")
	w := performRequest(r, "GET", "/auth/me", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestMe_Unauthenticated(t *testing.T) {
	r := setupTestRouter()
	w := performRequest(r, "GET", "/auth/me", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", w.Code)
	}
}

func TestListExercises(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "ex@example.com")
	w := performRequest(r, "GET", "/exercises", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var exercises []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&exercises)
	if len(exercises) == 0 {
		t.Error("Expected exercises to be seeded")
	}
}

func TestFilterExercisesByCategory(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "filter@example.com")
	w := performRequest(r, "GET", "/exercises?category=cardio", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var exercises []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&exercises)
	for _, e := range exercises {
		if e["category"] != "cardio" {
			t.Errorf("Expected cardio, got %s", e["category"])
		}
	}
}

func TestCreateWorkout_Success(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "wkcreate@example.com")
	body := map[string]interface{}{
		"title": "Push Day",
		"items": []map[string]interface{}{{"exercise_id": 1, "sets": 3, "reps": 10, "weight": 60.0}},
	}
	w := performRequest(r, "POST", "/workouts", body, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}
	var workout map[string]interface{}
	json.NewDecoder(w.Body).Decode(&workout)
	if workout["title"] == nil && workout["name"] == nil {
		t.Error("Expected workout title in response")
	}
}

func TestCreateWorkout_Unauthenticated(t *testing.T) {
	r := setupTestRouter()
	w := performRequest(r, "POST", "/workouts", map[string]interface{}{"title": "Test"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", w.Code)
	}
}

func TestListWorkouts(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "wklist@example.com")
	performRequest(r, "POST", "/workouts", map[string]interface{}{"title": "Workout 1"}, token)
	performRequest(r, "POST", "/workouts", map[string]interface{}{"title": "Workout 2"}, token)
	w := performRequest(r, "GET", "/workouts", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var workouts []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&workouts)
	if len(workouts) < 2 {
		t.Errorf("Expected at least 2 workouts, got %d", len(workouts))
	}
}

func TestGetWorkout_NotOwned(t *testing.T) {
	r := setupTestRouter()
	token1 := registerAndGetToken(r, "owner@example.com")
	token2 := registerAndGetToken(r, "other@example.com")
	cw := performRequest(r, "POST", "/workouts", map[string]interface{}{"title": "Private"}, token1)
	var workout map[string]interface{}
	json.NewDecoder(cw.Body).Decode(&workout)
	id := int(workout["id"].(float64))
	w := performRequest(r, "GET", fmt.Sprintf("/workouts/%d", id), nil, token2)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestUpdateWorkout(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "update@example.com")
	cw := performRequest(r, "POST", "/workouts", map[string]interface{}{"title": "Old Name"}, token)
	var workout map[string]interface{}
	json.NewDecoder(cw.Body).Decode(&workout)
	id := int(workout["id"].(float64))
	w := performRequest(r, "PUT", fmt.Sprintf("/workouts/%d", id), map[string]interface{}{"title": "New Name", "status": "active"}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestDeleteWorkout(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "delete@example.com")
	cw := performRequest(r, "POST", "/workouts", map[string]interface{}{"title": "To Delete"}, token)
	var workout map[string]interface{}
	json.NewDecoder(cw.Body).Decode(&workout)
	id := int(workout["id"].(float64))
	w := performRequest(r, "DELETE", fmt.Sprintf("/workouts/%d", id), nil, token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", w.Code)
	}
	gw := performRequest(r, "GET", fmt.Sprintf("/workouts/%d", id), nil, token)
	if gw.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 after delete, got %d", gw.Code)
	}
}

func TestGetReport(t *testing.T) {
	r := setupTestRouter()
	token := registerAndGetToken(r, "report@example.com")
	w := performRequest(r, "GET", "/workouts/report", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}
