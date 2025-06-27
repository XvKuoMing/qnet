package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool         *pgxpool.Pool
	listenerConn *pgx.Conn
}

type Task struct {
	ID       int    `json:"id"`
	BaseURL  string `json:"base_url"`
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Headers  string `json:"headers"`
	Body     string `json:"body"`
	Status   string `json:"status"`
	Response string `json:"response"`
}

func (task *Task) Resolve() ([]byte, error) {
	// simply run the task
	headers := http.Header{}
	err := json.Unmarshal([]byte(task.Headers), &headers)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	req, err := http.NewRequest(task.Method, task.BaseURL+task.Endpoint, bytes.NewBuffer([]byte(task.Body)))
	if err != nil {
		return nil, err
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return body, err
}

func InitDB(host string, port string, user string, password string, dbname string) *DB {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)

	// Create connection pool for general database operations
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}

	// Create separate connection for listening to notifications
	listenerConn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}

	db := &DB{
		pool:         pool,
		listenerConn: listenerConn,
	}
	return db
}

func (db *DB) CreateTask(task *Task) error {
	_, err := db.pool.Exec(context.Background(),
		"INSERT INTO tasks (base_url, endpoint, method, headers, body) VALUES ($1, $2, $3, $4, $5)",
		task.BaseURL, task.Endpoint, task.Method, task.Headers, task.Body)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) GetTaskByID(id int) (*Task, error) {
	var task Task
	err := db.pool.QueryRow(context.Background(),
		"SELECT id, base_url, endpoint, method, headers, body, status FROM tasks WHERE id = $1",
		id).Scan(&task.ID, &task.BaseURL, &task.Endpoint, &task.Method, &task.Headers, &task.Body, &task.Status)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (db *DB) MarkTaskResolved(taskID int, response string) error {
	_, err := db.pool.Exec(context.Background(),
		"UPDATE tasks SET status = 'resolved', response = $1 WHERE id = $2",
		response, taskID)
	return err
}

func (db *DB) ReinsertTask(taskID int) error {
	// Get the original task
	task, err := db.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	// Delete the original task
	_, err = db.pool.Exec(context.Background(), "DELETE FROM tasks WHERE id = $1", taskID)
	if err != nil {
		return err
	}

	// Insert a new task (this will trigger notification again)
	return db.CreateTask(task)
}

func (db *DB) ProcessTask(taskID int) error {
	task, err := db.GetTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task %d: %v", taskID, err)
	}

	if task.Status != "pending" {
		// Task is no longer pending, skip
		return nil
	}

	log.Printf("Processing task %d: %s %s%s", task.ID, task.Method, task.BaseURL, task.Endpoint)

	response, err := task.Resolve()
	if err != nil {
		log.Printf("Task %d failed: %v. Reinserting...", taskID, err)
		return db.ReinsertTask(taskID)
	}

	log.Printf("Task %d completed successfully", taskID)
	return db.MarkTaskResolved(taskID, string(response))
}

func (db *DB) StartListener() {
	// Start listening for notifications using the dedicated listener connection
	_, err := db.listenerConn.Exec(context.Background(), "LISTEN pending_task")
	if err != nil {
		log.Printf("Failed to start listening: %v", err)
		return
	}

	log.Println("Started listening for pending task notifications...")

	go func() {
		for {
			notification, err := db.listenerConn.WaitForNotification(context.Background())
			if err != nil {
				log.Printf("Error waiting for notification: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if notification.Channel == "pending_task" {
				log.Printf("Received notification for task: %s", notification.Payload)

				// Parse task ID from payload
				var taskID int
				_, err := fmt.Sscanf(notification.Payload, "%d", &taskID)
				if err != nil {
					log.Printf("Failed to parse task ID from notification: %v", err)
					continue
				}

				// Process the task
				err = db.ProcessTask(taskID)
				if err != nil {
					log.Printf("Failed to process task %d: %v", taskID, err)
				}
			}
		}
	}()
}

func (db *DB) Close() {
	db.pool.Close()
	db.listenerConn.Close(context.Background())
}
