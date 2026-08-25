package schoolyear

import (
	"testing"

	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestValidateTransitionMatrix(t *testing.T) {
	tests := []struct {
		name   string
		from   data.SchoolYearState
		to     data.SchoolYearState
		role   auth.OrganizationRole
		reason string
		want   error
	}{
		{name: "setup to active owner", from: data.SchoolYearSetup, to: data.SchoolYearActive, role: auth.RoleOwner},
		{name: "setup to active administrator", from: data.SchoolYearSetup, to: data.SchoolYearActive, role: auth.RoleAdministrator},
		{name: "active to closed administrator", from: data.SchoolYearActive, to: data.SchoolYearClosed, role: auth.RoleAdministrator},
		{name: "active to setup forbidden", from: data.SchoolYearActive, to: data.SchoolYearSetup, role: auth.RoleOwner, want: ErrInvalidTransition},
		{name: "setup to closed forbidden", from: data.SchoolYearSetup, to: data.SchoolYearClosed, role: auth.RoleOwner, want: ErrInvalidTransition},
		{name: "closed to active owner reason", from: data.SchoolYearClosed, to: data.SchoolYearActive, role: auth.RoleOwner, reason: "fix import"},
		{name: "closed to active administrator", from: data.SchoolYearClosed, to: data.SchoolYearActive, role: auth.RoleAdministrator, reason: "fix import", want: ErrOwnerRequired},
		{name: "closed to active without reason", from: data.SchoolYearClosed, to: data.SchoolYearActive, role: auth.RoleOwner, want: ErrReasonRequired},
		{name: "coordinator cannot transition", from: data.SchoolYearSetup, to: data.SchoolYearActive, role: auth.RoleCoordinator, want: ErrRoleRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTransition(test.from, test.to, test.role, test.reason)
			if test.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, test.want)
		})
	}
}
