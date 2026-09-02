package preference

import (
	"context"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

type accessCodeRecipientDetails struct {
	DisplayName string
	HomeroomID  ids.XID
	Homeroom    string
}

func accessCodeRecipient(ctx context.Context, tx *data.Tx, schoolYearID, studentID ids.XID) (accessCodeRecipientDetails, error) {
	student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
	if err != nil {
		return accessCodeRecipientDetails{}, err
	}
	givenName := student.LegalGivenName
	if student.PreferredGivenName != nil && *student.PreferredGivenName != "" {
		givenName = *student.PreferredGivenName
	}
	homeroom, err := tx.GetHomeroomByID(ctx, schoolYearID, student.HomeroomID)
	if err != nil {
		return accessCodeRecipientDetails{}, err
	}
	return accessCodeRecipientDetails{DisplayName: givenName + " " + student.LegalFamilyName, HomeroomID: student.HomeroomID, Homeroom: homeroom.Name}, nil
}
