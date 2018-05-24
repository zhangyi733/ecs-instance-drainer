package helper

import (
	"log"
	"os"
)

func EnvMustHave(key string) (value string) {
	if value = os.Getenv(key); value == "" {
		log.Fatalf("ENV %q is not set", key)
	}
	return value
}
