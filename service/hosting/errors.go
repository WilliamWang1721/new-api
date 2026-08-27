package hosting

import "fmt"

type ToolHandoffError struct {
	Reason string
	Err    error
}

func (e *ToolHandoffError) Error() string {
	if e == nil {
		return "hosting tool handoff"
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "hosting tool handoff"
}

func (e *ToolHandoffError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func handoffErr(reason, message string) error {
	return &ToolHandoffError{Reason: reason, Err: fmt.Errorf("%s", message)}
}
