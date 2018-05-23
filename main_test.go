package main

import (
	"os"
	"os/exec"
	"os/signal"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Ensure our envvars are not present
	os.Setenv("LIFECYCLE_QUEUE", "")
	os.Setenv("TERMINATION_TIMEOUT", "")
}

func TestGracefulStop(t *testing.T) {
	if os.Getenv("GRACEFUL_STOP") == "1" {
		sigc := make(chan os.Signal, 1)
		go func() {
			stop(sigc)
		}()
		defer signal.Stop(sigc)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestGracefulStop")
	cmd.Env = append(os.Environ(), "GRACEFUL_STOP=1")

	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	assert.Nil(t, err)
}
