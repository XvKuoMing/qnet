package receiver

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"qnet/db"

	"github.com/gin-gonic/gin"
)

type endpointMap map[string][]string // base_url: [endpoint1, endpoint2]

var register = map[string]string{} // endpoint: base_url, used for fast lookup
var recieverDB *db.DB

func Load() {
	// loads tasks.json
	tasks, err := os.ReadFile("tasks.json")
	if err != nil {
		log.Fatal(err)
	}
	var endpoints endpointMap
	err = json.Unmarshal(tasks, &endpoints)
	if err != nil {
		log.Fatal(err)
	}
	for baseURL, endpoints := range endpoints {
		for _, endpoint := range endpoints {
			register[endpoint] = baseURL
		}
	}
}

func Register(endpoint string) string {
	baseURL, ok := register[endpoint]
	if !ok {
		log.Println("BaseURL not found for endpoint", endpoint)
		return ""
	}
	return baseURL
}

func proxyHandler(c *gin.Context) {
	method := c.Request.Method
	endpoint := c.Request.URL.Path
	baseURL := Register(endpoint)
	headers := c.Request.Header
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}
	if baseURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "BaseURL not found for endpoint"})
		return
	}

	// Convert headers to JSON string
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		log.Printf("Error marshaling headers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process headers"})
		return
	}

	task := &db.Task{
		BaseURL:  baseURL,
		Endpoint: endpoint,
		Method:   method,
		Headers:  string(headersJSON),
		Body:     string(body),
	}

	err = recieverDB.CreateTask(task)
	if err != nil {
		log.Printf("Error creating task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	log.Printf("Created task: %s %s%s", method, baseURL, endpoint)
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func Serve(host string, port string, db *db.DB) {
	recieverDB = db
	r := gin.Default()
	r.Any("/*path", proxyHandler)
	log.Printf("Starting server on %s:%s", host, port)
	r.Run(host + ":" + port)
}
