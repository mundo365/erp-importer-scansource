package domain

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(message string) error {
	return &ValidationError{Message: message}
}

// PermanentError marks a business failure that must not requeue the message.
type PermanentError struct {
	Stage     string
	Code      string
	Message   string
	Cause     error
}

func (e *PermanentError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *PermanentError) Unwrap() error { return e.Cause }

func Permanent(stage, code, message string) *PermanentError {
	return &PermanentError{Stage: stage, Code: code, Message: message}
}

func PermanentWrap(stage, code, message string, cause error) *PermanentError {
	return &PermanentError{Stage: stage, Code: code, Message: message, Cause: cause}
}
