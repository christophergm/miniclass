package preference

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"context"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

// RankedChoiceAccessCode contains plaintext only at the moment a code is
// issued. The database stores only its hash.
type RankedChoiceAccessCode struct {
	StudentID ids.XID
	Code      string
}

// IssueRankedChoiceAccessCodes creates one high-entropy code per student. It
// is called inside the session lifecycle transaction so an opening cannot
// commit without its corresponding access grants.
func IssueRankedChoiceAccessCodes(ctx context.Context, tx *data.Tx, session data.Session, studentIDs []ids.XID) ([]RankedChoiceAccessCode, error) {
	if tx == nil {
		return nil, errors.New("issue ranked-choice access codes: transaction is nil")
	}
	if session.RankedChoice == nil {
		return nil, ErrRankedChoiceNotConfigured
	}
	ordered := append([]ids.XID(nil), studentIDs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	result := make([]RankedChoiceAccessCode, 0, len(ordered))
	var previous ids.XID
	for _, studentID := range ordered {
		if studentID == "" || studentID == previous {
			continue
		}
		previous = studentID
		code, err := newRankedChoiceCode()
		if err != nil {
			return nil, err
		}
		if _, err := tx.CreateRankedChoiceAccessCode(ctx, session.SchoolYearID, session.ProgramID, session.ID, studentID, rankedChoiceCodeHash(code)); err != nil {
			return nil, err
		}
		result = append(result, RankedChoiceAccessCode{StudentID: studentID, Code: code})
	}
	return result, nil
}

func newRankedChoiceCode() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate ranked-choice access code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func rankedChoiceCodeHash(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}
