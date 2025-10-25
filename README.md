# Go Task Queue with REST API

A production-ready task queue implementation in Go demonstrating concurrency patterns, graceful shutdown, and RESTful API design.

## 🎯 Project Overview

This project implements an asynchronous task processing system with a REST API frontend. It showcases core Go concurrency concepts including goroutines, channels, context, mutexes, and graceful shutdown patterns.

### Key Features

- ✅ **Concurrent Workers**: Multiple goroutines processing tasks in parallel
- ✅ **REST API**: Submit tasks and check status via HTTP endpoints
- ✅ **Graceful Shutdown**: Clean exit handling Ctrl+C and system signals
- ✅ **Task Timeouts**: Per-task timeout protection (5 seconds)
- ✅ **Thread-Safe Results**: Safe concurrent access using RWMutex
- ✅ **Context Cancellation**: Proper propagation of cancellation signals
- ✅ **Buffered Channels**: Handle traffic bursts efficiently

## 🏗️ Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP POST /submit
       ▼
┌─────────────────┐
│   REST API      │
│   Handler       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│   Task Queue    │─────▶│   Worker 1   │
│   (Channel)     │      └──────────────┘
│                 │      ┌──────────────┐
│  Buffered: 100  │─────▶│   Worker 2   │
│                 │      └──────────────┘
│                 │      ┌──────────────┐
│                 │─────▶│   Worker N   │
└─────────────────┘      └──────┬───────┘
                                │
                                ▼
                         ┌──────────────┐
                         │   Result     │
                         │  Collector   │
                         └──────────────┘
                                │
                                ▼
                         ┌──────────────┐
                         │   Results    │
                         │   Map        │
                         │ (Thread-safe)│
                         └──────────────┘
```

## 📚 Learning Concepts

This project demonstrates the following Go concepts:

### 1. Goroutines
Lightweight threads managed by Go's runtime. Each worker runs in its own goroutine.

```go
go worker(id, taskChan, resultChan, &wg)
```

### 2. Channels
Type-safe pipes for communication between goroutines.

```go
taskChan := make(chan Task, 100)  // Buffered channel
taskChan <- task                   // Send
task := <-taskChan                 // Receive
```

### 3. Select Statement
Multiplexing multiple channel operations.

```go
select {
case <-ctx.Done():
    // Handle cancellation
case task := <-taskChan:
    // Process task
}
```

### 4. Context
Cancellation and timeout propagation across goroutines.

```go
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()
```

### 5. WaitGroup
Synchronization primitive to wait for goroutines to complete.

```go
wg.Add(1)        // Increment counter
defer wg.Done()  // Decrement when done
wg.Wait()        // Block until counter is zero
```

### 6. Mutex (RWMutex)
Protecting shared data from race conditions.

```go
resultsMux.RLock()   // Read lock (multiple readers)
resultsMux.RUnlock()
resultsMux.Lock()    // Write lock (exclusive)
resultsMux.Unlock()
```

### 7. Graceful Shutdown
Clean termination handling OS signals and draining work.

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
```

## 🚀 Getting Started

### Prerequisites

- Go 1.16 or higher
- Basic understanding of HTTP and REST APIs

### Installation

1. Clone or create the project:
```bash
mkdir task-queue && cd task-queue
```

2. Create `main.go` with the provided code

3. Initialize Go module:
```bash
go mod init task-queue
go mod tidy
```

### Running the Server

```bash
go run main.go
```

You should see:
```
🚀 Task Queue API Server starting on :8080
📝 Endpoints:
   POST   /submit       - Submit a new task
   GET    /status/{id}  - Get task status
   GET    /stats        - Get queue statistics

💡 Press Ctrl+C for graceful shutdown
```

## 📖 API Documentation

### Submit a Task

**Endpoint:** `POST /submit`

**Request Body:**
```json
{
  "id": "optional-task-id",
  "type": "email",
  "payload": "Send welcome email to user@example.com"
}
```

**Response:**
```json
{
  "message": "Task submitted successfully",
  "task_id": "task-1698234567890",
  "status": "pending"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"type": "email", "payload": "Send notification"}'
```

### Get Task Status

**Endpoint:** `GET /status/{task_id}`

**Response (Pending):**
```json
{
  "task_id": "task-1698234567890",
  "status": "pending or not found"
}
```

**Response (Completed):**
```json
{
  "task_id": "task-1698234567890",
  "success": true,
  "data": "Task 'email' processed by worker 2 in 0.75s",
  "processed_at": "2024-10-25T10:30:45Z"
}
```

**Response (Failed):**
```json
{
  "task_id": "task-1698234567890",
  "success": false,
  "error": "Worker 3 failed to process task",
  "processed_at": "2024-10-25T10:30:45Z"
}
```

**Example:**
```bash
curl http://localhost:8080/status/task-1698234567890
```

### Get Queue Statistics

**Endpoint:** `GET /stats`

**Response:**
```json
{
  "total_processed": 150,
  "successful": 120,
  "failed": 30,
  "queue_length": 5
}
```

**Example:**
```bash
curl http://localhost:8080/stats
```

## 🧪 Testing Examples

### Submit Multiple Tasks

```bash
# Submit 20 tasks
for i in {1..20}; do
  curl -X POST http://localhost:8080/submit \
    -H "Content-Type: application/json" \
    -d "{\"type\": \"process\", \"payload\": \"work-$i\"}"
  echo ""
done
```

### Check Task Status

```bash
# Get the task_id from submit response, then:
TASK_ID="task-1698234567890"
curl http://localhost:8080/status/$TASK_ID
```

### Monitor Queue

```bash
# Watch stats in real-time (requires 'watch' command)
watch -n 1 'curl -s http://localhost:8080/stats | jq'
```

### Test Graceful Shutdown

1. Start the server
2. Submit multiple tasks
3. Press `Ctrl+C`
4. Observe clean shutdown in logs

## 🔍 How It Works

### Task Submission Flow

1. Client sends POST request to `/submit`
2. API handler generates task ID if not provided
3. Task is sent to buffered task channel
4. Handler returns immediately (async)
5. Available worker picks up task
6. Worker processes task (with 5-second timeout)
7. Result is sent to result channel
8. Result collector stores result in map
9. Client can poll `/status/{id}` to get result

### Worker Lifecycle

1. Worker starts, waits on task channel
2. `select` statement monitors:
   - Task channel for new work
   - Context for shutdown signal
3. On task received:
   - Process with timeout context
   - Send result or handle timeout
4. On shutdown signal:
   - Finish current task
   - Exit gracefully

### Graceful Shutdown Sequence

1. OS sends SIGINT (Ctrl+C) or SIGTERM
2. Signal handler triggers
3. HTTP server stops accepting new connections
4. Existing HTTP requests complete
5. Context cancellation propagates to workers
6. Task channel closes (no new tasks)
7. Workers finish current tasks
8. WaitGroup waits (max 10 seconds)
9. Result channel closes
10. Program exits cleanly

## 🎓 Learning Path

If you're new to Go concurrency, study these files in order:

1. **Channels** - Basic communication (`taskChan`, `resultChan`)
2. **Goroutines** - Concurrent execution (`go worker()`)
3. **WaitGroups** - Synchronization (`wg.Wait()`)
4. **Mutex** - Shared state protection (`resultsMux`)
5. **Select** - Channel multiplexing (`select { case... }`)
6. **Context** - Cancellation propagation (`ctx.Done()`)
7. **Signals** - OS signal handling (`signal.Notify()`)

## 🛠️ Configuration

Current configuration (modify in `main.go`):

```go
numWorkers := 5              // Number of concurrent workers
taskChanSize := 100          // Task queue buffer size
resultChanSize := 100        // Result queue buffer size
taskTimeout := 5*time.Second // Per-task timeout
shutdownTimeout := 10*time.Second // Graceful shutdown timeout
serverPort := ":8080"        // HTTP server port
```

## 🐛 Troubleshooting

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080
# Kill it
kill -9 <PID>
```

### Workers Not Processing
- Check if task channel is full (increase buffer size)
- Verify workers are started before sending tasks
- Check logs for panic or errors

### Shutdown Hangs
- Increase shutdown timeout
- Check for blocking operations in workers
- Ensure all channels are properly closed

## 📈 Performance Considerations

- **Buffer sizes**: Adjust based on traffic patterns
- **Worker count**: Match to CPU cores for CPU-bound tasks, more for I/O-bound
- **Timeouts**: Set based on expected task duration
- **Result storage**: Consider TTL or external storage for production

## 🚧 Potential Enhancements

- [ ] Persistent storage (Redis/PostgreSQL for results)
- [ ] Priority queues (high/medium/low priority tasks)
- [ ] Retry logic with exponential backoff
- [ ] Dead letter queue for failed tasks
- [ ] Metrics and monitoring (Prometheus)
- [ ] Task scheduling (cron-like functionality)
- [ ] Worker pools with dynamic scaling
- [ ] Distributed task queue (multiple servers)
- [ ] WebSocket for real-time status updates
- [ ] Authentication and rate limiting

## 📝 License

This is a learning project - use it however you like!

## 🤝 Contributing

This is an educational project. Feel free to fork and experiment!

## 📚 Additional Resources

- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Context Package](https://pkg.go.dev/context)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example - Goroutines](https://gobyexample.com/goroutines)

---

Built with ❤️ to learn Go concurrency patterns.
