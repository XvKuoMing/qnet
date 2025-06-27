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
		log.Println("No .env file found, assuming environment variables are set")
	}
	receiver.Load()

	// receiver host and port
	host := os.Getenv("GO_HOST")
	port := os.Getenv("GO_PORT")

	// database host and port
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	database := db.InitDB(dbHost, dbPort, dbUser, dbPassword, dbName)

	// Start the notification listener
	database.StartListener()

	receiver.Serve(host, port, database)
}
