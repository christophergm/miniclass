package integration

import (
	"fmt"
	"os"
	"testing"

	testharness "github.com/chrismott/miniclass/internal/testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if err := testharness.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
