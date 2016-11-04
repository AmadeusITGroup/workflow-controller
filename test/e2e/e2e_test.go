package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}

func TestMain(m *testing.M) {
	fmt.Printf("TestMain\n")
	flag.Parse()
	os.Exit(m.Run())
}
