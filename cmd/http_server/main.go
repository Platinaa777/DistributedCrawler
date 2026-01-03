package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const serverAddr = "127.0.0.1:8080"

type CrawlJob struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type CrawlTask struct {
	ID         string    `json:"id"`
	JobID      string    `json:"job_id"`
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

type PageSnapshot struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"task_id"`
	URL         string    `json:"url"`
	HTTPStatus  int       `json:"http_status"`
	ContentType string    `json:"content_type"`
	StorageKey  string    `json:"storage_key"`
	FetchedAt   time.Time `json:"fetched_at"`
}

type ExtractedRecord struct {
	ID        string                 `json:"id"`
	TaskID    string                 `json:"task_id"`
	SourceURL string                 `json:"source_url"`
	Data      map[string]interface{} `json:"data"`
	ParsedAt  time.Time              `json:"parsed_at"`
}

type mockStore struct {
	jobs      map[string]CrawlJob
	tasks     map[string]CrawlTask
	snapshots map[string]PageSnapshot
	records   map[string]ExtractedRecord
}

func newMockStore() *mockStore {
	createdAt := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	enqueuedAt := time.Date(2025, 12, 22, 12, 5, 0, 0, time.UTC)
	fetchedAt := time.Date(2025, 12, 22, 12, 10, 0, 0, time.UTC)
	parsedAt := time.Date(2025, 12, 22, 12, 15, 0, 0, time.UTC)

	job := CrawlJob{
		ID:        "uuid1",
		Name:      "Example Crawl Job",
		Status:    "Running",
		CreatedAt: createdAt,
	}
	task := CrawlTask{
		ID:         "uuid2",
		JobID:      job.ID,
		URL:        "http://example.com",
		Status:     "Pending",
		EnqueuedAt: enqueuedAt,
	}
	snapshot := PageSnapshot{
		ID:          "uuid3",
		TaskID:      task.ID,
		URL:         "http://example.com",
		HTTPStatus:  200,
		ContentType: "text/html",
		StorageKey:  "example-com-page-1",
		FetchedAt:   fetchedAt,
	}
	record := ExtractedRecord{
		ID:        "uuid4",
		TaskID:    task.ID,
		SourceURL: "http://example.com",
		Data: map[string]interface{}{
			"title":   "Example Page",
			"content": "This is a test page",
		},
		ParsedAt: parsedAt,
	}

	return &mockStore{
		jobs:      map[string]CrawlJob{job.ID: job},
		tasks:     map[string]CrawlTask{task.ID: task},
		snapshots: map[string]PageSnapshot{snapshot.ID: snapshot},
		records:   map[string]ExtractedRecord{record.ID: record},
	}
}

func (s *mockStore) listJobs() []CrawlJob {
	jobs := make([]CrawlJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	return jobs
}

func (s *mockStore) listTasksByJob(jobID string) []CrawlTask {
	tasks := make([]CrawlTask, 0)
	for _, t := range s.tasks {
		if t.JobID == jobID {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

func (s *mockStore) listSnapshotsByTask(taskID string) []PageSnapshot {
	items := make([]PageSnapshot, 0)
	for _, sn := range s.snapshots {
		if sn.TaskID == taskID {
			items = append(items, sn)
		}
	}
	return items
}

func (s *mockStore) listRecordsByTask(taskID string) []ExtractedRecord {
	items := make([]ExtractedRecord, 0)
	for _, rec := range s.records {
		if rec.TaskID == taskID {
			items = append(items, rec)
		}
	}
	return items
}

func main() {
	store := newMockStore()
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "distributed-crawler HTTP server is up")
	})

	r.Route("/jobs", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, store.listJobs())
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if payload.Name == "" {
				writeError(w, http.StatusBadRequest, "name is required")
				return
			}

			id := newID("job")
			job := CrawlJob{
				ID:        id,
				Name:      payload.Name,
				Status:    defaultValue(payload.Status, "Pending"),
				CreatedAt: time.Now().UTC(),
			}
			store.jobs[id] = job
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, job)
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			job, ok := store.jobs[id]
			if !ok {
				writeError(w, http.StatusNotFound, "job not found")
				return
			}
			writeJSON(w, job)
		})
		r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			job, ok := store.jobs[id]
			if !ok {
				writeError(w, http.StatusNotFound, "job not found")
				return
			}
			var payload struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if payload.Status != "" {
				job.Status = payload.Status
				if payload.Status == "Completed" || payload.Status == "Finished" {
					now := time.Now().UTC()
					job.CompletedAt = &now
				}
			}
			store.jobs[id] = job
			writeJSON(w, job)
		})
	})

	r.Route("/tasks", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				JobID  string `json:"job_id"`
				URL    string `json:"url"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if payload.JobID == "" || payload.URL == "" {
				writeError(w, http.StatusBadRequest, "job_id and url are required")
				return
			}
			if _, ok := store.jobs[payload.JobID]; !ok {
				writeError(w, http.StatusNotFound, "parent job not found")
				return
			}

			id := newID("task")
			task := CrawlTask{
				ID:         id,
				JobID:      payload.JobID,
				URL:        payload.URL,
				Status:     defaultValue(payload.Status, "Pending"),
				EnqueuedAt: time.Now().UTC(),
			}
			store.tasks[id] = task
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, task)
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			task, ok := store.tasks[id]
			if !ok {
				writeError(w, http.StatusNotFound, "task not found")
				return
			}
			writeJSON(w, task)
		})
		r.Get("/job/{job_id}", func(w http.ResponseWriter, r *http.Request) {
			jobID := chi.URLParam(r, "job_id")
			if _, ok := store.jobs[jobID]; !ok {
				writeError(w, http.StatusNotFound, "job not found")
				return
			}
			writeJSON(w, store.listTasksByJob(jobID))
		})
		r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			task, ok := store.tasks[id]
			if !ok {
				writeError(w, http.StatusNotFound, "task not found")
				return
			}
			var payload struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if payload.Status != "" {
				task.Status = payload.Status
			}
			store.tasks[id] = task
			writeJSON(w, task)
		})
	})

	r.Route("/snapshots", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				TaskID      string `json:"task_id"`
				URL         string `json:"url"`
				HTTPStatus  int    `json:"http_status"`
				ContentType string `json:"content_type"`
				StorageKey  string `json:"storage_key"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if payload.TaskID == "" || payload.URL == "" {
				writeError(w, http.StatusBadRequest, "task_id and url are required")
				return
			}
			if _, ok := store.tasks[payload.TaskID]; !ok {
				writeError(w, http.StatusNotFound, "task not found")
				return
			}

			id := newID("snapshot")
			item := PageSnapshot{
				ID:          id,
				TaskID:      payload.TaskID,
				URL:         payload.URL,
				HTTPStatus:  payload.HTTPStatus,
				ContentType: payload.ContentType,
				StorageKey:  payload.StorageKey,
				FetchedAt:   time.Now().UTC(),
			}
			store.snapshots[id] = item
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, item)
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			item, ok := store.snapshots[id]
			if !ok {
				writeError(w, http.StatusNotFound, "snapshot not found")
				return
			}
			writeJSON(w, item)
		})
		r.Get("/task/{task_id}", func(w http.ResponseWriter, r *http.Request) {
			taskID := chi.URLParam(r, "task_id")
			if _, ok := store.tasks[taskID]; !ok {
				writeError(w, http.StatusNotFound, "task not found")
				return
			}
			writeJSON(w, store.listSnapshotsByTask(taskID))
		})
	})

	r.Route("/records", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				TaskID    string                 `json:"task_id"`
				SourceURL string                 `json:"source_url"`
				Data      map[string]interface{} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if payload.TaskID == "" || payload.SourceURL == "" || payload.Data == nil {
				writeError(w, http.StatusBadRequest, "task_id, source_url and data are required")
				return
			}
			if _, ok := store.tasks[payload.TaskID]; !ok {
				writeError(w, http.StatusNotFound, "task not found")
				return
			}

			id := newID("record")
			record := ExtractedRecord{
				ID:        id,
				TaskID:    payload.TaskID,
				SourceURL: payload.SourceURL,
				Data:      payload.Data,
				ParsedAt:  time.Now().UTC(),
			}
			store.records[id] = record
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, record)
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			record, ok := store.records[id]
			if !ok {
				writeError(w, http.StatusNotFound, "record not found")
				return
			}
			writeJSON(w, record)
		})
		r.Get("/task/{task_id}", func(w http.ResponseWriter, r *http.Request) {
			taskID := chi.URLParam(r, "task_id")
			if _, ok := store.tasks[taskID]; !ok {
				writeError(w, http.StatusNotFound, "task not found")
				return
			}
			writeJSON(w, store.listRecordsByTask(taskID))
		})
	})

	log.Printf("starting HTTP server on http://%s", serverAddr)
	err := http.ListenAndServe(serverAddr, logRequests(r))
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func defaultValue(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
