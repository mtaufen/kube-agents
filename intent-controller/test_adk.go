package main

import (
	"log"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
)

func main() {
	a, err := llmagent.New(llmagent.Config{
		Name:        "test_agent",
		Description: "A test agent.",
		Instruction: "You are a test agent. Say 'Hello'.",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	r, err := runner.New(runner.Config{
		Agent: a,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}
	_ = r
	log.Println("Runner created successfully")
}
