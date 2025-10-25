package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID   int
	Name string
}

type Result struct {
	TaskID  int
	Success bool
	Data    string
	Error   error
}

func main() {
	// Create channels
	taskQueue := make(chan Task, 10)
	resultQueue := make(chan Result, 10) // Channel to collect results

	// WaitGroup for workers
	var wg sync.WaitGroup

	// Start 3 worker goroutines
	numWorkers := 3
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, taskQueue, resultQueue, &wg)
	}

	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go resultCollector(resultQueue, &collectorWg)

	// Sending tasks to the queue
	numTasks := 10
	for i := 1; i <= numTasks; i++ {
		task := Task{
			ID:   i,
			Name: fmt.Sprintf("Task-%d", i),
		}
		fmt.Printf("📥 Sending %s to queue\n", task.Name)
		taskQueue <- task
	}

	close(taskQueue)

	fmt.Println("\n⏳ Waiting for workers to finish...")
	wg.Wait()
	fmt.Println("✅ All workers done!")

	close(resultQueue)

	collectorWg.Wait()
	fmt.Println("📊 Result collection complete!")
}

func worker(id int, taskQueue <-chan Task, resultQueue chan<- Result, wg *sync.WaitGroup) {
	// chan<- Result means "send-only channel" - worker can only write results
	defer wg.Done()

	for task := range taskQueue {
		fmt.Printf("   🔧 Worker %d started processing %s\n", id, task.Name)

		// Simulate work that might fail
		time.Sleep(500 * time.Millisecond)

		// Randomly succeed or fail (30% chance of failure)
		success := rand.Float32() > 0.3

		result := Result{
			TaskID:  task.ID,
			Success: success,
		}

		if success {
			result.Data = fmt.Sprintf("Processed by worker %d", id)
			fmt.Printf("   ✓ Worker %d finished %s successfully\n", id, task.Name)
		} else {
			result.Error = fmt.Errorf("worker %d failed to process task", id)
			fmt.Printf("   ❌ Worker %d failed %s\n", id, task.Name)
		}

		// Send result to the result queue
		resultQueue <- result
	}

	fmt.Printf("👷 Worker %d shutting down\n", id)
}

// resultCollector gathers all results and prints a summary
func resultCollector(resultQueue <-chan Result, wg *sync.WaitGroup) {
	defer wg.Done()

	var successCount, failCount int
	var results []Result

	// Collect all results
	for result := range resultQueue {
		results = append(results, result)
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	// Print summary
	separator := strings.Repeat("=", 50)
	fmt.Println("\n" + separator)
	fmt.Println("📈 RESULTS SUMMARY")
	fmt.Println(separator)
	fmt.Printf("Total tasks: %d\n", len(results))
	fmt.Printf("✓ Successful: %d\n", successCount)
	fmt.Printf("❌ Failed: %d\n", failCount)
	fmt.Println(separator)

	if failCount > 0 {
		fmt.Println("\n❌ Failed tasks:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("   Task %d: %v\n", result.TaskID, result.Error)
			}
		}
	}
}
