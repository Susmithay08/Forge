package database

import (
	"database/sql"
	"fmt"
	"time"
	"workout-tracker/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS exercises (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT NOT NULL,
		muscle_group TEXT
	);

	CREATE TABLE IF NOT EXISTS workouts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT DEFAULT '',
		comment TEXT DEFAULT '',
		status TEXT DEFAULT 'pending',
		scheduled_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS workout_exercises (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workout_id INTEGER NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
		exercise_id INTEGER NOT NULL REFERENCES exercises(id),
		sets INTEGER DEFAULT 0,
		reps INTEGER DEFAULT 0,
		weight_kg REAL DEFAULT 0,
		duration_sec INTEGER DEFAULT 0,
		notes TEXT DEFAULT ''
	);
	`
	_, err := db.Exec(schema)
	return err
}

// ---- Users ----

func (db *DB) CreateUser(name, email, hash string) (*models.User, error) {
	res, err := db.Exec(`INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)`, name, email, hash)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return db.GetUserByID(id)
}

func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRow(`SELECT id, name, email, password_hash, created_at FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) GetUserByID(id int64) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRow(`SELECT id, name, email, password_hash, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// ---- Exercises ----

func (db *DB) GetExercises() ([]models.Exercise, error) {
	rows, err := db.Query(`SELECT id, name, description, category, muscle_group FROM exercises ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Exercise
	for rows.Next() {
		var e models.Exercise
		rows.Scan(&e.ID, &e.Name, &e.Description, &e.Category, &e.MuscleGroup)
		list = append(list, e)
	}
	return list, nil
}

func (db *DB) GetExerciseByID(id int64) (*models.Exercise, error) {
	e := &models.Exercise{}
	err := db.QueryRow(`SELECT id, name, description, category, muscle_group FROM exercises WHERE id = ?`, id).
		Scan(&e.ID, &e.Name, &e.Description, &e.Category, &e.MuscleGroup)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return e, err
}

// ---- Workouts ----

func (db *DB) CreateWorkout(userID int64, req models.CreateWorkoutRequest) (*models.Workout, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var scheduledStr interface{}
	if req.ScheduledAt != nil {
		scheduledStr = req.ScheduledAt.Format(time.RFC3339)
	}

	res, err := tx.Exec(`INSERT INTO workouts (user_id, title, description, scheduled_at, status) VALUES (?, ?, ?, ?, 'pending')`,
		userID, req.Title, req.Description, scheduledStr)
	if err != nil {
		return nil, err
	}
	wid, _ := res.LastInsertId()

	for _, e := range req.Exercises {
		_, err := tx.Exec(`INSERT INTO workout_exercises (workout_id, exercise_id, sets, reps, weight_kg, duration_sec, notes) VALUES (?,?,?,?,?,?,?)`,
			wid, e.ExerciseID, e.Sets, e.Reps, e.WeightKg, e.DurationSec, e.Notes)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return db.GetWorkoutByID(wid, userID)
}

func (db *DB) GetWorkoutByID(id, userID int64) (*models.Workout, error) {
	w := &models.Workout{}
	var scheduledStr, completedStr sql.NullString
	err := db.QueryRow(`SELECT id, user_id, title, description, comment, status, scheduled_at, completed_at, created_at, updated_at FROM workouts WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.Comment, &w.Status, &scheduledStr, &completedStr, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if scheduledStr.Valid {
		t, _ := time.Parse(time.RFC3339, scheduledStr.String)
		w.ScheduledAt = &t
	}
	if completedStr.Valid {
		t, _ := time.Parse(time.RFC3339, completedStr.String)
		w.CompletedAt = &t
	}
	w.Exercises, _ = db.getWorkoutExercises(id)
	return w, nil
}

func (db *DB) getWorkoutExercises(workoutID int64) ([]models.WorkoutExercise, error) {
	rows, err := db.Query(`
		SELECT we.id, we.workout_id, we.exercise_id, we.sets, we.reps, we.weight_kg, we.duration_sec, we.notes,
		       e.id, e.name, e.description, e.category, e.muscle_group
		FROM workout_exercises we
		JOIN exercises e ON e.id = we.exercise_id
		WHERE we.workout_id = ?`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.WorkoutExercise
	for rows.Next() {
		var we models.WorkoutExercise
		e := &models.Exercise{}
		rows.Scan(&we.ID, &we.WorkoutID, &we.ExerciseID, &we.Sets, &we.Reps, &we.WeightKg, &we.DurationSec, &we.Notes,
			&e.ID, &e.Name, &e.Description, &e.Category, &e.MuscleGroup)
		we.Exercise = e
		list = append(list, we)
	}
	return list, nil
}

func (db *DB) ListWorkouts(userID int64, status string) ([]models.Workout, error) {
	query := `SELECT id, user_id, title, description, comment, status, scheduled_at, completed_at, created_at, updated_at FROM workouts WHERE user_id = ?`
	args := []interface{}{userID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY COALESCE(scheduled_at, created_at) ASC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Workout
	for rows.Next() {
		var w models.Workout
		var scheduledStr, completedStr sql.NullString
		rows.Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.Comment, &w.Status, &scheduledStr, &completedStr, &w.CreatedAt, &w.UpdatedAt)
		if scheduledStr.Valid {
			t, _ := time.Parse(time.RFC3339, scheduledStr.String)
			w.ScheduledAt = &t
		}
		if completedStr.Valid {
			t, _ := time.Parse(time.RFC3339, completedStr.String)
			w.CompletedAt = &t
		}
		w.Exercises, _ = db.getWorkoutExercises(w.ID)
		list = append(list, w)
	}
	return list, nil
}

func (db *DB) UpdateWorkout(id, userID int64, req models.UpdateWorkoutRequest) (*models.Workout, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	existing, err := db.GetWorkoutByID(id, userID)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("workout not found")
	}

	title := existing.Title
	desc := existing.Description
	comment := existing.Comment
	status := existing.Status
	var scheduledStr interface{}
	if existing.ScheduledAt != nil {
		scheduledStr = existing.ScheduledAt.Format(time.RFC3339)
	}

	if req.Title != nil {
		title = *req.Title
	}
	if req.Description != nil {
		desc = *req.Description
	}
	if req.Comment != nil {
		comment = *req.Comment
	}
	if req.Status != nil {
		status = *req.Status
	}
	if req.ScheduledAt != nil {
		scheduledStr = req.ScheduledAt.Format(time.RFC3339)
	}

	var completedStr interface{}
	if status == "completed" {
		completedStr = time.Now().Format(time.RFC3339)
	}

	_, err = tx.Exec(`UPDATE workouts SET title=?, description=?, comment=?, status=?, scheduled_at=?, completed_at=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`,
		title, desc, comment, status, scheduledStr, completedStr, id, userID)
	if err != nil {
		return nil, err
	}

	if req.Exercises != nil {
		_, err = tx.Exec(`DELETE FROM workout_exercises WHERE workout_id = ?`, id)
		if err != nil {
			return nil, err
		}
		for _, e := range req.Exercises {
			_, err := tx.Exec(`INSERT INTO workout_exercises (workout_id, exercise_id, sets, reps, weight_kg, duration_sec, notes) VALUES (?,?,?,?,?,?,?)`,
				id, e.ExerciseID, e.Sets, e.Reps, e.WeightKg, e.DurationSec, e.Notes)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return db.GetWorkoutByID(id, userID)
}

func (db *DB) DeleteWorkout(id, userID int64) error {
	res, err := db.Exec(`DELETE FROM workouts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workout not found")
	}
	return nil
}

func (db *DB) GetReport(userID int64) (*models.WorkoutReport, error) {
	report := &models.WorkoutReport{}

	db.QueryRow(`SELECT COUNT(*) FROM workouts WHERE user_id = ?`, userID).Scan(&report.TotalWorkouts)
	db.QueryRow(`SELECT COUNT(*) FROM workouts WHERE user_id = ? AND status = 'completed'`, userID).Scan(&report.CompletedWorkouts)

	var totalVol sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(we.sets * we.reps * we.weight_kg)
		FROM workout_exercises we
		JOIN workouts w ON w.id = we.workout_id
		WHERE w.user_id = ? AND w.status = 'completed'`, userID).Scan(&totalVol)
	if totalVol.Valid {
		report.TotalVolumeKg = totalVol.Float64
	}

	// avg per week
	var firstDate sql.NullString
	db.QueryRow(`SELECT MIN(created_at) FROM workouts WHERE user_id = ?`, userID).Scan(&firstDate)
	if firstDate.Valid && report.TotalWorkouts > 0 {
		t, _ := time.Parse("2006-01-02 15:04:05", firstDate.String)
		weeks := time.Since(t).Hours() / 168
		if weeks < 1 {
			weeks = 1
		}
		report.AvgWorkoutsPerWeek = float64(report.TotalWorkouts) / weeks
	}

	// most used exercise
	var exName sql.NullString
	db.QueryRow(`
		SELECT e.name FROM workout_exercises we
		JOIN exercises e ON e.id = we.exercise_id
		JOIN workouts w ON w.id = we.workout_id
		WHERE w.user_id = ?
		GROUP BY we.exercise_id ORDER BY COUNT(*) DESC LIMIT 1`, userID).Scan(&exName)
	if exName.Valid {
		report.MostUsedExercise = exName.String
	}

	workouts, _ := db.ListWorkouts(userID, "completed")
	report.Workouts = workouts

	return report, nil
}
