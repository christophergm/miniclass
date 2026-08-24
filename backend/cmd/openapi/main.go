// Command openapi writes the generated MiniClass OpenAPI contract.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chrismott/miniclass/internal/api"
)

func main() {
	output := flag.String("output", defaultOutputPath(), "path to the generated OpenAPI JSON document")
	flag.Parse()

	if err := write(*output); err != nil {
		fmt.Fprintf(os.Stderr, "generate OpenAPI: %v\n", err)
		os.Exit(1)
	}
}

func defaultOutputPath() string {
	if _, err := os.Stat("go.mod"); err == nil {
		return "openapi.json"
	}
	return filepath.Join("backend", "openapi.json")
}

func write(output string) error {
	document, err := json.MarshalIndent(api.NewOpenAPI(api.RouterOptions{}), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	document = append(document, '\n')
	if err := os.WriteFile(output, document, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}
	return nil
}
