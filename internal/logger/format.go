package logger

import "fmt"

func StepFormat(step, total int, message string) string {
	if step <= 0 || total <= 0 || step > total {
		return message
	}
	return fmt.Sprintf("Step %d/%d: %s", step, total, message)
}
