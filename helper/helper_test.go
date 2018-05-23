package helper

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvMustHaveNoValue(t *testing.T) {
	if os.Getenv("TEST_NO_VALUE") == "1" {
		EnvMustHave("TEST")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestEnvMustHaveNoValue")
	cmd.Env = append(os.Environ(), "TEST_NO_VALUE=1")

	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	assert.Error(t, err)
}

func TestEnvMustHaveValue(t *testing.T) {
	os.Setenv("LIFECYCLE_QUEUE", "test-queue")
	defer os.Setenv("LIFECYCLE_QUEUE", "")

	value := EnvMustHave("LIFECYCLE_QUEUE")
	assert.Equal(t, "test-queue", value)
}
