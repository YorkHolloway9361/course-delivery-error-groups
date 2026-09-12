package deliveryerrors

import (
	"fmt"
	"time"
)

// DeliveryFailure is the domain input received from a course delivery worker.
type DeliveryFailure struct {
	EventID       string    `json:"event_id"`
	CourseID      string    `json:"course_id"`
	LearnerID     string    `json:"learner_id"`
	DeliveryStage string    `json:"delivery_stage"`
	Deadline      time.Time `json:"deadline"`
	Exception     string    `json:"exception"`
}

// CaptureDecision records the compliance-relevant classification sent upstream.
type CaptureDecision struct {
	Title       string            `json:"title"`
	Message     string            `json:"message"`
	Level       string            `json:"level"`
	Fingerprint []string          `json:"fingerprint"`
	Exception   string            `json:"exception"`
	Context     map[string]string `json:"context"`
	EventID     string            `json:"-"`
}

// Classify groups operational repeats while preserving learner context for educator review.
func Classify(f DeliveryFailure, now time.Time) (CaptureDecision, error) {
	if f.EventID == "" || f.CourseID == "" || f.LearnerID == "" || f.DeliveryStage == "" || f.Deadline.IsZero() || f.Exception == "" {
		return CaptureDecision{}, fmt.Errorf("all delivery failure fields are required")
	}

	level := "warning"
	if !f.Deadline.After(now.Add(24 * time.Hour)) {
		level = "error"
	}

	return CaptureDecision{
		Title:       fmt.Sprintf("course delivery failed at %s", f.DeliveryStage),
		Message:     fmt.Sprintf("learner delivery failed for course %s", f.CourseID),
		Level:       level,
		Fingerprint: []string{f.CourseID, f.DeliveryStage},
		Exception:   f.Exception,
		Context: map[string]string{
			"course_id":      f.CourseID,
			"learner_id":     f.LearnerID,
			"delivery_stage": f.DeliveryStage,
			"deadline":       f.Deadline.UTC().Format(time.RFC3339),
		},
		EventID: f.EventID,
	}, nil
}
