package logger

import "fmt"

func StepFormat(step, total int, message string) string {
	return fmt.Sprintf("Step %d/%d: %s", step, total, message)
}
