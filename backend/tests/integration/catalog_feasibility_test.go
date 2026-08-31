package integration

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestCatalogFeasibilityUsesSessionParticipantsAndNeverBlocksCatalogWrites(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "catalog feasibility integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic catalog feasibility year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-catalog-grade", "Synthetic Catalog Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Catalog Room")
	require.NoError(t, err)
	first, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "First", LegalFamilyName: "Catalog", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	second, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Second", LegalFamilyName: "Catalog", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Catalog Program")
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, first.ID)
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, second.ID)
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Catalog Session", 1, []time.Time{time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	area, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic Catalog Area")
	require.NoError(t, err)
	service := program.New(harness.Database)

	capacity := 1
	_, err = factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Tagged offering", "Description", nil, capacity, grade.ID, grade.ID, "Room", "Point", "Instructions", &area.ID)
	require.NoError(t, err)
	feasibility, err := service.GetCatalogFeasibility(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, 2, feasibility.ParticipantCount)
	require.Equal(t, []string{program.CatalogCapacityShortWarning}, warningIDs(feasibility))

	// A catalog warning is advice, not a gate. The organizer can continue
	// authoring even while the current snapshot reports insufficient capacity.
	_, err = factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Untyped offering", "Description", nil, 1, grade.ID, grade.ID, "Room", "Point", "Instructions", nil)
	require.NoError(t, err)

	_, err = factory.CreateSessionNonParticipation(ctx, year.ID, programRow.ID, session.ID, second.ID, "synthetic absence")
	require.NoError(t, err)
	feasibility, err = service.GetCatalogFeasibility(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, 1, feasibility.ParticipantCount)
	require.NotContains(t, warningIDs(feasibility), program.CatalogCapacityShortWarning)
	require.Contains(t, warningIDs(feasibility), program.CatalogUnmatchedOfferingWarning)

	otherOrganization := harness.MintOrganization(t)
	_, err = service.GetCatalogFeasibility(ctx, string(otherOrganization), year.ID, programRow.ID, session.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func warningIDs(value program.CatalogFeasibility) []string {
	result := make([]string, 0, len(value.Warnings))
	for _, warning := range value.Warnings {
		result = append(result, warning.ID)
	}
	return result
}
