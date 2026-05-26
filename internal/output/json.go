package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/soumeet96/ecsdig/internal/model"
)

func PrintJSON(result *model.DiagnosisResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
	}
}
