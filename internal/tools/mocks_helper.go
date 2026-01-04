package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	_ = os.RemoveAll("mocks")
	if err := os.MkdirAll("mocks", 0o755); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("go", "run", "github.com/gojuno/minimock/v3/cmd/minimock@v3.4.7",
		"-i", "CrawlJobService",
		"-o", "./mocks/",
		"-s", "_minimock.go",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
