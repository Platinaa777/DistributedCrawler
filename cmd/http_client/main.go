package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	baseUrl = "http://localhost:8080"
	addJobs = "/jobs"
)

func main() {
	t := struct {
		Name   string
		Status string
	}{
		Name:   "Example",
		Status: "active",
	}

	data, err := json.Marshal(t)
	if err != nil {
		log.Fatalf("%v", err)
	}

	resp, err := http.Post(baseUrl+addJobs, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("%v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println(response)
}
