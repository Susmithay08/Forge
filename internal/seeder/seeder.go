package seeder

import (
	"workout-tracker/internal/database"
)

type Exercise struct {
	Name        string
	Description string
	Category    string
	MuscleGroup string
}

var exercises = []Exercise{
	// Strength - Chest
	{"Bench Press", "Classic compound chest exercise with barbell", "strength", "chest"},
	{"Push-Up", "Bodyweight chest and tricep exercise", "strength", "chest"},
	{"Incline Dumbbell Press", "Upper chest focused press", "strength", "chest"},
	{"Cable Fly", "Isolation chest exercise using cables", "strength", "chest"},
	// Strength - Back
	{"Pull-Up", "Compound back and bicep bodyweight exercise", "strength", "back"},
	{"Deadlift", "Full body compound lift targeting posterior chain", "strength", "back"},
	{"Bent Over Row", "Barbell row for mid and upper back", "strength", "back"},
	{"Lat Pulldown", "Machine exercise targeting lats", "strength", "back"},
	// Strength - Legs
	{"Squat", "King of lower body exercises", "strength", "legs"},
	{"Leg Press", "Machine compound leg exercise", "strength", "legs"},
	{"Romanian Deadlift", "Hamstring focused hip hinge", "strength", "legs"},
	{"Lunges", "Unilateral leg exercise for quads and glutes", "strength", "legs"},
	{"Calf Raise", "Isolation exercise for calves", "strength", "legs"},
	// Strength - Shoulders
	{"Overhead Press", "Compound shoulder pressing movement", "strength", "shoulders"},
	{"Lateral Raise", "Isolation for medial deltoid", "strength", "shoulders"},
	{"Front Raise", "Isolation for anterior deltoid", "strength", "shoulders"},
	{"Face Pull", "Rear delt and rotator cuff exercise", "strength", "shoulders"},
	// Strength - Arms
	{"Bicep Curl", "Isolation exercise for biceps", "strength", "arms"},
	{"Tricep Dips", "Compound tricep exercise", "strength", "arms"},
	{"Hammer Curl", "Brachialis and bicep curl variation", "strength", "arms"},
	{"Skull Crusher", "Tricep isolation with EZ bar", "strength", "arms"},
	// Cardio
	{"Running", "Steady state or interval outdoor run", "cardio", "full body"},
	{"Cycling", "Stationary or outdoor bike cardio", "cardio", "legs"},
	{"Jump Rope", "High intensity cardio with rope", "cardio", "full body"},
	{"Rowing Machine", "Full body cardio on rowing machine", "cardio", "full body"},
	{"Elliptical", "Low impact full body cardio", "cardio", "full body"},
	{"Burpees", "High intensity full body cardio", "cardio", "full body"},
	{"Box Jump", "Explosive plyometric exercise", "cardio", "legs"},
	// Flexibility
	{"Yoga Flow", "Dynamic stretching and flexibility routine", "flexibility", "full body"},
	{"Hip Flexor Stretch", "Static stretch for hip flexors", "flexibility", "hips"},
	{"Hamstring Stretch", "Static stretch for hamstrings", "flexibility", "legs"},
	{"Shoulder Mobility", "Shoulder rotation and mobility drills", "flexibility", "shoulders"},
	{"Pigeon Pose", "Deep hip opener yoga pose", "flexibility", "hips"},
	{"Foam Rolling", "Self-myofascial release technique", "flexibility", "full body"},
}

func SeedExercises(db *database.DB) error {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM exercises").Scan(&count)
	if count > 0 {
		return nil // already seeded
	}

	stmt, err := db.Prepare(`INSERT INTO exercises (name, description, category, muscle_group) VALUES (?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range exercises {
		if _, err := stmt.Exec(e.Name, e.Description, e.Category, e.MuscleGroup); err != nil {
			return err
		}
	}
	return nil
}
