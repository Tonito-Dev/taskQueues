package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Task represents a unit of work
type Task struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// Result represents the outcome of processing a task
type Result struct {
	TaskID      string    `json:"task_id"`
	Success     bool      `json:"success"`
	Data        string    `json:"data,omitempty"`
	Error       string    `json:"error,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}

// TaskQueue manages our task processing system
type TaskQueue struct {
	taskChan   chan Task
	resultChan chan Result
	results    map[string]Result // Store results by task ID
	resultsMux sync.RWMutex      // Protects the results map
	wg         sync.WaitGroup
}

// NewTaskQueue creates and starts a new task queue
func NewTaskQueue(numWorkers int) *TaskQueue {
	tq := &TaskQueue{
		taskChan:   make(chan Task, 100),
		resultChan: make(chan Result, 100),
		results:    make(map[string]Result),
	}

	// Start workers
	for i := 1; i <= numWorkers; i++ {
		tq.wg.Add(1)
		go tq.worker(i)
	}

	// Start result collector
	go tq.collectResults()

	return tq
}

// SubmitTask adds a task to the queue
func (tq *TaskQueue) SubmitTask(task Task) {
	log.Printf("📥 Received task: %s (type: %s)", task.ID, task.Type)
	tq.taskChan <- task
}

// GetResult retrieves the result of a task by ID
func (tq *TaskQueue) GetResult(taskID string) (Result, bool) {
	tq.resultsMux.RLock()         // Read lock
	defer tq.resultsMux.RUnlock() // Unlock when done
	result, exists := tq.results[taskID]
	return result, exists
}

// worker processes tasks from the queue
func (tq *TaskQueue) worker(id int) {
	defer tq.wg.Done()

	for task := range tq.taskChan {
		log.Printf("   🔧 Worker %d processing task %s", id, task.ID)

		// Simulate different types of work
		processingTime := time.Duration(500+rand.Intn(1000)) * time.Millisecond
		time.Sleep(processingTime)

		// Simulate success/failure (20% failure rate)
		success := rand.Float32() > 0.2

		result := Result{
			TaskID:      task.ID,
			Success:     success,
			ProcessedAt: time.Now(),
		}

		if success {
			result.Data = fmt.Sprintf("Task '%s' processed by worker %d in %.2fs",
				task.Type, id, processingTime.Seconds())
			log.Printf("   ✓ Worker %d completed task %s", id, task.ID)
		} else {
			result.Error = fmt.Sprintf("Worker %d failed to process task", id)
			log.Printf("   ❌ Worker %d failed task %s", id, task.ID)
		}

		// Send result to collector
		tq.resultChan <- result
	}

	log.Printf("👷 Worker %d shutting down", id)
}

// collectResults gathers results and stores them
func (tq *TaskQueue) collectResults() {
	for result := range tq.resultChan {
		tq.resultsMux.Lock() // Write lock
		tq.results[result.TaskID] = result
		tq.resultsMux.Unlock()
	}
}

// Global task queue instance
var taskQueue *TaskQueue

func main() {
	// Initialize task queue with 5 workers
	taskQueue = NewTaskQueue(5)

	// Setup HTTP routes
	http.HandleFunc("/submit", submitTaskHandler)
	http.HandleFunc("/status/", getStatusHandler)
	http.HandleFunc("/stats", getStatsHandler)

	// Start server
	fmt.Println("🚀 Task Queue API Server starting on :8080")
	fmt.Println("📝 Endpoints:")
	fmt.Println("   POST   /submit       - Submit a new task")
	fmt.Println("   GET    /status/{id}  - Get task status")
	fmt.Println("   GET    /stats        - Get queue statistics")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// HTTP Handler: Submit a new task
func submitTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	task.CreatedAt = time.Now()

	// Submit to queue
	taskQueue.SubmitTask(task)

	// Return immediate response
	response := map[string]string{
		"message": "Task submitted successfully",
		"task_id": task.ID,
		"status":  "pending",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HTTP Handler: Get task status
func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from URL path
	taskID := strings.TrimPrefix(r.URL.Path, "/status/")
	if taskID == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	// Check if result exists
	result, exists := taskQueue.GetResult(taskID)

	w.Header().Set("Content-Type", "application/json")

	if !exists {
		response := map[string]string{
			"task_id": taskID,
			"status":  "pending or not found",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return result
	json.NewEncoder(w).Encode(result)
}

// HTTP Handler: Get queue statistics
func getStatsHandler(w http.ResponseWriter, r *http.Request) {
	taskQueue.resultsMux.RLock()
	totalTasks := len(taskQueue.results)

	var successCount, failCount int
	for _, result := range taskQueue.results {
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}
	taskQueue.resultsMux.RUnlock()

	stats := map[string]interface{}{
		"total_processed": totalTasks,
		"successful":      successCount,
		"failed":          failCount,
		"queue_length":    len(taskQueue.taskChan),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
