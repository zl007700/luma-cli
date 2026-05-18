package cloud

import (
	"time"
)

// WaitTaskComplete polls a task until it completes or times out.
// Returns (status, stillRunning=true if timed out).
func WaitTaskComplete(taskID string, cardKey string, timeoutSec int) (map[string]any, bool) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	pollInterval := 2

	for time.Now().Before(deadline) {
		status, err := PollTask(taskID, cardKey)
		if err != nil {
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		taskStatus := ""
		if s, ok := status["status"].(string); ok {
			taskStatus = s
		}
		if taskStatus == "completed" || taskStatus == "failed" {
			return status, false
		}
		time.Sleep(time.Duration(pollInterval) * time.Second)
	}

	// Timed out — return last known status
	finalStatus, _ := PollTask(taskID, cardKey)
	return finalStatus, true
}