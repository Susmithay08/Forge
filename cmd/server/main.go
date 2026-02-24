package main

import (
	"log"
	"net/http"
	"os"
	"workout-tracker/internal/database"
	"workout-tracker/internal/handlers"
	"workout-tracker/internal/middleware"
	"workout-tracker/internal/seeder"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	} else {
		log.Println("Loaded config from .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "workout_tracker.db"
	}

	db, err := database.New(dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	if err := db.Migrate(); err != nil {
		log.Fatal("Migration failed:", err)
	}
	if err := seeder.SeedExercises(db); err != nil {
		log.Fatal("Seeding failed:", err)
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.Static("/app", "./frontend")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	authH := handlers.NewAuthHandler(db)
	exerciseH := handlers.NewExerciseHandler(db)
	workoutH := handlers.NewWorkoutHandler(db)

	// Config endpoint - serves Groq key to authenticated users only
	r.GET("/api/config", middleware.AuthRequired(), func(c *gin.Context) {
		groqKey := os.Getenv("GROQ_API_KEY")
		c.JSON(http.StatusOK, gin.H{
			"groq_key_set": groqKey != "",
			"groq_key":     groqKey,
		})
	})

	auth := r.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.GET("/me", middleware.AuthRequired(), authH.Me)
	}

	r.GET("/exercises", middleware.AuthRequired(), exerciseH.List)

	workouts := r.Group("/workouts", middleware.AuthRequired())
	{
		workouts.POST("", workoutH.Create)
		workouts.GET("", workoutH.List)
		workouts.GET("/report", workoutH.Report)
		workouts.GET("/:id", workoutH.Get)
		workouts.PUT("/:id", workoutH.Update)
		workouts.DELETE("/:id", workoutH.Delete)
	}

	log.Printf("Workout Tracker running on http://localhost:%s\n", port)
	r.Run(":" + port)
}
