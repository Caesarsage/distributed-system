package task

import "fmt"

// String implements Stringer interface
func (s Status) String() string {
	return string(s)
}

// IsTerminal returns true if status is final
func (s Status) IsTerminal() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusCanceled
}

// Validate checks if status is valid
func (s Status) Validate() error {
	switch s {
	case StatusPending, StatusRunning, StatusSuccess, StatusFailed, StatusCanceled, StatusRetrying:
		return nil
	default:
		return fmt.Errorf("invalid task status: %s", s)
	}
}
