package cmd

// SilentError marks an error whose user-facing output has already been written.
type SilentError struct {
	Err error
}

func (err SilentError) Error() string { return err.Err.Error() }
func (err SilentError) Unwrap() error { return err.Err }
func (err SilentError) Silent() bool  { return true }
