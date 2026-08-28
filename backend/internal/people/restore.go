package people

import (
	"errors"
	"strings"
)

var (
	ErrRestoreReasonRequired = errors.New("restore reason is required")
	ErrRestoreNotDeleted     = errors.New("roster record is not deleted")
)

func restoreReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", ErrRestoreReasonRequired
	}
	return reason, nil
}
