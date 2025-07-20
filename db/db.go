package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

var DB *sql.DB

func InitDB() {
	_ = godotenv.Load()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("❌ DATABASE_URL not found in .env")
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("❌ Failed to connect to DB:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("❌ DB not reachable:", err)
	}

	log.Println("✅ Connected to PostgreSQL")
}
