package main

import (
	"fmt"
	"testing"
)

func TestConfig(t *testing.T) {
	config := ReadConfig()
	fmt.Println("Configured server is: ", config.Server)
	if config.Server == "" {

		t.Error("expected server name string, got", config.Server)
	}
}
