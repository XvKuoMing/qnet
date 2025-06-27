package main

import (
	"log"
	"os"
	"qnet/db"
	"qnet/receiver"

	"github.com/joho/godotenv"
)

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	receiver.Load()

	// receiver host and port
	host := os.Getenv("GO_HOST")
	port := os.Getenv("GO_PORT")

	// database host and port
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	database := db.InitDB(dbHost, dbPort, dbUser, dbPassword, dbName)

	// Start the notification listener
	database.StartListener()

	receiver.Serve(host, port, database)
}
