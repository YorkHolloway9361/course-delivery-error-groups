package deliveryerrors

import (
	"testing"
	"time"
)

func TestClassifyDeadlineRisk(t *testing.T) {
	now := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		deadline  time.Time
		wantLevel string
	}{
		{name: "deadline inside review window", deadline: now.Add(6 * time.Hour), wantLevel: "error"},
		{name: "deadline at review boundary", deadline: now.Add(24 * time.Hour), wantLevel: "error"},
		{name: "deadline outside review window", deadline: now.Add(72 * time.Hour), wantLevel: "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Classify(DeliveryFailure{
				EventID: "delivery-evt-1042", CourseID: "course-ledger-7", LearnerID: "learner-18",
				DeliveryStage: "assignment-release", Deadline: tt.deadline, Exception: "queue publish rejected",
			}, now)
			if err != nil {
				t.Fatal(err)
			}
			if got.Level != tt.wantLevel {
				t.Fatalf("level = %q, want %q", got.Level, tt.wantLevel)
			}
			if got.Fingerprint[0] != "course-ledger-7" || got.Fingerprint[1] != "assignment-release" {
				t.Fatalf("fingerprint = %v", got.Fingerprint)
			}
		})
	}
}
