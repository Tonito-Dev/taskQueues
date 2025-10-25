package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

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
	results    map[string]Result
	resultsMux sync.RWMutex
	wg         sync.WaitGroup
	ctx        context.Context    // For cancellation
	cancel     context.CancelFunc // Function to trigger cancellation
}

// NewTaskQueue creates and starts a new task queue
func NewTaskQueue(numWorkers int) *TaskQueue {
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	tq := &TaskQueue{
		taskChan:   make(chan Task, 100),
		resultChan: make(chan Result, 100),
		results:    make(map[string]Result),
		ctx:        ctx,
		cancel:     cancel,
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
func (tq *TaskQueue) SubmitTask(task Task) error {
	// Check if we're shutting down
	select {
	case <-tq.ctx.Done():
		return fmt.Errorf("queue is shutting down")
	case tq.taskChan <- task:
		log.Printf(" Received task: %s (type: %s)", task.ID, task.Type)
		return nil
	}
}

// GetResult retrieves the result of a task by ID
func (tq *TaskQueue) GetResult(taskID string) (Result, bool) {
	tq.resultsMux.RLock()
	defer tq.resultsMux.RUnlock()
	result, exists := tq.results[taskID]
	return result, exists
}

// Shutdown gracefully stops the task queue
func (tq *TaskQueue) Shutdown(timeout time.Duration) error {
	log.Println(" Starting graceful shutdown...")

	// Signal all goroutines to stop
	tq.cancel()

	// Close task channel (no new tasks accepted)
	close(tq.taskChan)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		tq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println(" All workers finished")
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout: workers didn't finish in time")
	}

	// Close result channel
	close(tq.resultChan)

	log.Println(" Graceful shutdown complete")
	return nil
}

// worker processes tasks from the queue with context awareness
func (tq *TaskQueue) worker(id int) {
	defer tq.wg.Done()

	for {
		select {
		case <-tq.ctx.Done():
			// Context cancelled - shutdown signal
			log.Printf(" Worker %d received shutdown signal", id)
			return

		case task, ok := <-tq.taskChan:
			if !ok {
				// Channel closed
				log.Printf(" Worker %d: task channel closed", id)
				return
			}

			// Process task with timeout
			result := tq.processTaskWithTimeout(id, task, 5*time.Second)

			// Try to send result, but respect shutdown
			select {
			case tq.resultChan <- result:
			case <-tq.ctx.Done():
				log.Printf("  Worker %d: discarding result due to shutdown", id)
				return
			}
		}
	}
}

// processTaskWithTimeout processes a task with a timeout
func (tq *TaskQueue) processTaskWithTimeout(workerID int, task Task, timeout time.Duration) Result {
	log.Printf("    Worker %d processing task %s", workerID, task.ID)

	// Create a context with timeout for this specific task
	ctx, cancel := context.WithTimeout(tq.ctx, timeout)
	defer cancel()

	// Channel to receive result from processing
	resultChan := make(chan Result, 1)

	// Do the actual work in a goroutine
	go func() {
		processingTime := time.Duration(500+rand.Intn(1000)) * time.Millisecond

		// Simulate work that respects cancellation
		select {
		case <-time.After(processingTime):
			// Work completed
			success := rand.Float32() > 0.2
			result := Result{
				TaskID:      task.ID,
				Success:     success,
				ProcessedAt: time.Now(),
			}

			if success {
				result.Data = fmt.Sprintf("Task '%s' processed by worker %d in %.2fs",
					task.Type, workerID, processingTime.Seconds())
				log.Printf("   ✓ Worker %d completed task %s", workerID, task.ID)
			} else {
				result.Error = fmt.Sprintf("Worker %d failed to process task", workerID)
				log.Printf("    Worker %d failed task %s", workerID, task.ID)
			}

			resultChan <- result

		case <-ctx.Done():
			// Work was cancelled or timed out
			return
		}
	}()

	// Wait for result or timeout/cancellation
	select {
	case result := <-resultChan:
		return result

	case <-ctx.Done():
		// Task timed out or was cancelled
		log.Printf("     Worker %d: task %s timed out or cancelled", workerID, task.ID)
		return Result{
			TaskID:      task.ID,
			Success:     false,
			Error:       "Task timed out or was cancelled",
			ProcessedAt: time.Now(),
		}
	}
}

// collectResults gathers results and stores them
func (tq *TaskQueue) collectResults() {
	for result := range tq.resultChan {
		tq.resultsMux.Lock()
		tq.results[result.TaskID] = result
		tq.resultsMux.Unlock()
	}
	log.Println(" Result collector stopped")
}

// Global task queue instance
var taskQueue *TaskQueue

func main() {
	// Initialize task queue with 5 workers
	taskQueue = NewTaskQueue(5)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Setup HTTP routes
	http.HandleFunc("/submit", submitTaskHandler)
	http.HandleFunc("/status/", getStatusHandler)
	http.HandleFunc("/stats", getStatsHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	// Start server in a goroutine
	go func() {
		fmt.Println(" Task Queue API Server starting on :8080")
		fmt.Println(" Endpoints:")
		fmt.Println("   POST   /submit       - Submit a new task")
		fmt.Println("   GET    /status/{id}  - Get task status")
		fmt.Println("   GET    /stats        - Get queue statistics")
		fmt.Println("\n Press Ctrl+C for graceful shutdown")

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	log.Println("\n Interrupt signal received")

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	log.Println(" HTTP server stopped")

	// Shutdown task queue with timeout
	if err := taskQueue.Shutdown(10 * time.Second); err != nil {
		log.Printf("Task queue shutdown error: %v", err)
	}

	log.Println(" Goodbye!")
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

	// Submit to queue (might fail if shutting down)
	if err := taskQueue.SubmitTask(task); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

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
	taskID := strings.TrimPrefix(r.URL.Path, "/status/")
	if taskID == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

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

	stats := map[string]any{
		"total_processed": totalTasks,
		"successful":      successCount,
		"failed":          failCount,
		"queue_length":    len(taskQueue.taskChan),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
