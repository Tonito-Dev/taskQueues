package main

import (
	"fmt"
	"sync"
	"time"
)

// Task represents a unit of work
type Task struct {
	ID   int
	Name string
}

func main() {
	// Create a channel to hold tasks
	// This is like a pipe where we can send and receive tasks
	taskQueue := make(chan Task, 10)

	// Create a waitgroup to track when all workers are down
	// This is like a counter that workers will decrement when they finish
	var wg sync.WaitGroup

	// Start 3 worker goroutines
	// Each worker will process tasks concurrently
	numWorkers := 3
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, taskQueue, &wg)
	}

	// send tasks to the queue
	// These will be picked up by available workers
	for i := 1; i <= 10; i++ {
		task := Task{
			ID:   i,
			Name: fmt.Sprintf("Task-%d", i),
		}
		fmt.Printf("📥 Sending %s to queue\n", task.Name)
		taskQueue <- task // sending task to channel
	}

	// close the channel to signals no more tasks are coming
	close(taskQueue)

	// Wait for all workers to finish
	// This blocks until the counter reaches zero
	fmt.Println("\n⏳ Waiting for workers to finish...")
	wg.Wait() // Giving workers time to finish processing

	fmt.Println("\n✅ All workers done!")
}

func worker(id int, taskQueue <-chan Task, wg *sync.WaitGroup) {
	// The <-chan Task means this channel is receive-only
	// workers can only read from it, not send to it
	defer wg.Done()

	for task := range taskQueue {
		//'range' will keep reading from the channel until it's close
		fmt.Printf("   🔧 Worker %d started processing %s\n", id, task.Name)

		// Simulate work by sleeping
		time.Sleep(500 * time.Millisecond)

		fmt.Printf("   ✓ Worker %d finished %s\n", id, task.Name)
	}
	fmt.Printf("👷 Worker %d shutting down\n", id)
}
