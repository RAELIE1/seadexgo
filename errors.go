package seadex

import "fmt"

type SeaDexError struct {
	Message string
}

func (e *SeaDexError) Error() string {
	return e.Message
}

type EntryNotFoundError struct {
	SeaDexError
}

func newEntryNotFoundError(format string, args ...any) *EntryNotFoundError {
	return &EntryNotFoundError{SeaDexError{Message: fmt.Sprintf(format, args...)}}
}

type BadBackupFileError struct {
	SeaDexError
}

func newBadBackupFileError(format string, args ...any) *BadBackupFileError {
	return &BadBackupFileError{SeaDexError{Message: fmt.Sprintf(format, args...)}}
}
