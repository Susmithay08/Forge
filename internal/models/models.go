package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Exercise struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	MuscleGroup string `json:"muscle_group"`
}

type Workout struct {
	ID          int64             `json:"id"`
	UserID      int64             `json:"user_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Comment     string            `json:"comment"`
	Status      string            `json:"status"`
	ScheduledAt *time.Time        `json:"scheduled_at"`
	CompletedAt *time.Time        `json:"completed_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Exercises   []WorkoutExercise `json:"exercises,omitempty"`
}

type WorkoutExercise struct {
	ID          int64     `json:"id"`
	WorkoutID   int64     `json:"workout_id"`
	ExerciseID  int64     `json:"exercise_id"`
	Sets        int       `json:"sets"`
	Reps        int       `json:"reps"`
	WeightKg    float64   `json:"weight_kg"`
	DurationSec int       `json:"duration_sec"`
	Notes       string    `json:"notes"`
	Exercise    *Exercise `json:"exercise,omitempty"`
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateWorkoutRequest struct {
	Title       string                   `json:"title" binding:"required"`
	Description string                   `json:"description"`
	ScheduledAt *time.Time               `json:"scheduled_at"`
	Exercises   []WorkoutExerciseRequest `json:"exercises"`
}

type WorkoutExerciseRequest struct {
	ExerciseID  int64   `json:"exercise_id" binding:"required"`
	Sets        int     `json:"sets"`
	Reps        int     `json:"reps"`
	WeightKg    float64 `json:"weight_kg"`
	DurationSec int     `json:"duration_sec"`
	Notes       string  `json:"notes"`
}

type UpdateWorkoutRequest struct {
	Title       *string                  `json:"title"`
	Description *string                  `json:"description"`
	Comment     *string                  `json:"comment"`
	Status      *string                  `json:"status"`
	ScheduledAt *time.Time               `json:"scheduled_at"`
	Exercises   []WorkoutExerciseRequest `json:"exercises"`
}

type WorkoutReport struct {
	TotalWorkouts      int       `json:"total_workouts"`
	CompletedWorkouts  int       `json:"completed_workouts"`
	TotalVolumeKg      float64   `json:"total_volume_kg"`
	AvgWorkoutsPerWeek float64   `json:"avg_workouts_per_week"`
	MostUsedExercise   string    `json:"most_used_exercise"`
	Workouts           []Workout `json:"workouts"`
}
