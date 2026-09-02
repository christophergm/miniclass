package preference

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"context"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

// RankedChoiceAccessCode contains plaintext only at the moment a code is
// issued. The database stores only its hash.
type RankedChoiceAccessCode struct {
	StudentID   ids.XID
	Code        string
	DisplayName string
	HomeroomID  ids.XID
	Homeroom    string
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
		recipient, err := accessCodeRecipient(ctx, tx, session.SchoolYearID, studentID)
		if err != nil {
			return nil, err
		}
		result = append(result, RankedChoiceAccessCode{StudentID: studentID, Code: code, DisplayName: recipient.DisplayName, HomeroomID: recipient.HomeroomID, Homeroom: recipient.Homeroom})
	}
	return result, nil
}

// RegenerateRankedChoiceAccessCodes revokes every active grant and returns a
// fresh plaintext code for each participating student. Plaintext is returned
// only from this issuance operation; the database retains only hashes.
func (s *Service) RegenerateRankedChoiceAccessCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, reason string) ([]RankedChoiceAccessCode, error) {
	if s == nil || s.database == nil {
		return nil, ErrPreferenceServiceNil
	}
	if strings.TrimSpace(reason) == "" {
		return nil, ErrAccessCodeReasonRequired
	}
	var result []RankedChoiceAccessCode
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if session.RankedChoice == nil || session.State != data.SessionVotingOpen || session.RankedChoice.Deadline == nil || !time.Now().UTC().Before(*session.RankedChoice.Deadline) {
			return ErrRankedChoiceNotAccepting
		}
		if _, err := tx.RevokeRankedChoiceAccessCodes(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		students, err := tx.ListParticipatingStudentIDs(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		result, err = IssueRankedChoiceAccessCodes(ctx, tx, session, students)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionRankedChoiceCodeChange, ObjectType: "ranked_choice_access_code", ObjectID: &sessionID, SchoolYearID: &schoolYearID, Reason: strings.TrimSpace(reason), ChangeSummary: mustJSON(map[string]any{"regenerated": len(result)})})
	})
	if err != nil {
		return nil, fmt.Errorf("regenerate ranked-choice access codes: %w", err)
	}
	return result, nil
}

func (s *Service) RevokeRankedChoiceAccessCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, reason string) error {
	if s == nil || s.database == nil {
		return ErrPreferenceServiceNil
	}
	if strings.TrimSpace(reason) == "" {
		return ErrAccessCodeReasonRequired
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		count, err := tx.RevokeRankedChoiceAccessCodes(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionRankedChoiceCodeChange, ObjectType: "ranked_choice_access_code", ObjectID: &sessionID, SchoolYearID: &schoolYearID, Reason: strings.TrimSpace(reason), ChangeSummary: mustJSON(map[string]any{"revoked": count})})
	})
	if err != nil {
		return fmt.Errorf("revoke ranked-choice access codes: %w", err)
	}
	return nil
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
