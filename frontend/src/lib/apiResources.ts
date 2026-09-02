import { api, unwrap, unwrapList, unwrapNoContent } from "./api";
import type { components } from "./api.generated";

// Typed wrappers over the one client in ./api. Every type here is derived from
// the generated contract rather than restated, so a backend field rename fails
// `tsc` instead of a page (ADR 0004).

type Schemas = components["schemas"];

export type HealthResponse = Schemas["HealthResponse"];
export type MeResponse = Schemas["MeResponse"];
export type SchoolYear = Schemas["SchoolYearResponse"];
export type SchoolYearState = SchoolYear["state"];
export type GradeLevel = Schemas["GradeLevelOutput"];
export type Homeroom = Schemas["HomeroomOutput"];
export type VocabularyResponse = Schemas["VocabularyResponse"];
export type Administrator = Schemas["AdministratorResponse"];
export type AdministratorInvitation = Schemas["InvitationResponse"];
export type AuditLogResponse = Schemas["AuditLogOutputBody"];
export type AuditLogEntry = Schemas["AuditLogEntry"];
export type ImportPreview = Schemas["Preview"];
export type ImportKind = "roster_json" | "grades_csv";
export type Program = Schemas["ProgramResponse"];
export type ProgramMembership = Schemas["ProgramMembershipResponse"];
export type ProgramRosterSummary = Schemas["ProgramRosterSummaryResponse"];
export type InterestArea = Schemas["InterestAreaResponse"];
export type Session = Schemas["SessionResponse"];
export type SessionTransition = Schemas["SessionTransitionResponse"];
export type CatalogFeasibility = Schemas["CatalogFeasibilityResponse"];
export type CatalogFeasibilityWarning = Schemas["CatalogFeasibilityWarningResponse"];
export type MeetingDate = Schemas["MeetingDateResponse"];
export type Offering = Schemas["OfferingResponse"];
export type SessionNonParticipation = Schemas["SessionNonParticipationResponse"];
export type ObjectiveWeights = Schemas["ObjectiveWeightsResponse"];
export type ObjectiveWeightsInput = Schemas["ObjectiveWeightsInput"];
export type ObjectiveWeightOverrides = Schemas["ObjectiveWeightOverridesResponse"];
export type ProgramObjectiveWeights = Schemas["ProgramObjectiveWeightsResponse"];
export type SessionObjectiveWeights = Schemas["SessionObjectiveWeightsResponse"];
export type InterestProfileSurvey = Schemas["InterestProfileSurveyResponse"];
export type InterestProfileSurveyInput = Schemas["InterestProfileSurveyInputBody"];
export type InterestProfileSurveyTransitionInput =
  Schemas["TransitionInterestProfileSurveyInputBody"];
export type InterestProfileSurveyTransition = Schemas["InterestProfileSurveyTransitionResponse"];
export type InterestProfileSurveyCodes = Schemas["InterestProfileSurveyCodeResponse"];
export type RankedChoiceAccessCode = Schemas["RankedChoiceAccessCodeResponse"];
export type PreferenceForm = Schemas["PreferenceFormResponse"];
export type GuardianPreferenceForms = Schemas["GuardianPreferenceFormsResponse"];
export type PreferenceInterestAnswerInput = Schemas["InterestProfileAnswerInput"];
export type PreferenceRankedAnswerInput = Schemas["RankedChoiceAnswerInput"];
export type AdultOTPRequest = Schemas["RequestAdultOTPOutputBody"];
export type GuardianSession = Schemas["GuardianSessionResponse"];
export type MFAEnrollment = Schemas["MFAEnrollmentOutputBody"];
export type AdministrativeSession = Schemas["AdministrativeSessionOutputBody"];
export type AdultAccountLink = Schemas["AdultAccountLinkResponse"];

export const resourceApi = {
  getHealth: () => unwrap(api.GET("/api/health")),

  getMe: () => unwrap(api.GET("/api/me")),
  claimInvitation: (token: string) => unwrap(api.POST("/api/auth/claim", { body: { token } })),
  requestAdultOTP: (organizationID: string, schoolYearID: string, email: string) =>
    unwrap(
      api.POST("/api/auth/adult/otp/request", {
        body: { organization_id: organizationID, school_year_id: schoolYearID, email },
      }),
    ),
  verifyAdultOTP: (challengeID: string, code: string) =>
    unwrap(
      api.POST("/api/auth/adult/otp/verify", {
        body: { challenge_id: challengeID, code },
      }),
    ),
  getGuardianAuthContext: () => unwrap(api.GET("/api/auth/guardian")),
  enrollMFA: () => unwrap(api.POST("/api/auth/mfa/enroll", {})),
  verifyMFA: (code?: string, recoveryCode?: string) =>
    unwrap(api.POST("/api/auth/mfa/verify", { body: { code, recovery_code: recoveryCode } })),
  enterGuardianMode: (schoolYearID: string) =>
    unwrap(api.POST("/api/auth/mode/guardian", { body: { school_year_id: schoolYearID } })),
  listAdultAccountLinks: (schoolYearID: string) =>
    unwrapList(
      api.GET("/api/auth/adult-account-links", {
        params: { query: { school_year_id: schoolYearID } },
      }),
    ),
  createAdultAccountLink: (schoolYearID: string, adultID: string, userID: string) =>
    unwrap(
      api.POST("/api/auth/adult-account-links", {
        body: { school_year_id: schoolYearID, adult_id: adultID, user_id: userID },
      }),
    ),
  deleteAdultAccountLink: (linkID: string, schoolYearID: string) =>
    unwrapNoContent(
      api.DELETE("/api/auth/adult-account-links/{linkID}", {
        params: { path: { linkID }, query: { school_year_id: schoolYearID } },
      }),
    ),
  revokeAuthSession: () => unwrapNoContent(api.POST("/api/auth/session/revoke", {})),

  getAuditLog: (options: { objectType?: string; cursor?: string; limit?: number } = {}) =>
    unwrap(
      api.GET("/api/audit-log", {
        params: {
          query: {
            object_type: options.objectType || undefined,
            cursor: options.cursor || undefined,
            limit: options.limit,
          },
        },
      }),
    ),

  listSchoolYears: () => unwrapList(api.GET("/api/school-years")),
  getSchoolYear: (schoolYearID: string) =>
    unwrap(api.GET("/api/school-years/{schoolYearID}", { params: { path: { schoolYearID } } })),
  createSchoolYear: (label: string) => unwrap(api.POST("/api/school-years", { body: { label } })),
  updateSchoolYear: (schoolYearID: string, update: Schemas["UpdateSchoolYearInputBody"]) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}", {
        params: { path: { schoolYearID } },
        body: update,
      }),
    ),

  listPrograms: (schoolYearID: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/programs", { params: { path: { schoolYearID } } }),
    ),
  createProgram: (schoolYearID: string, name: string) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/programs", {
        params: { path: { schoolYearID } },
        body: { name },
      }),
    ),
  listStudents: (schoolYearID: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/students", {
        params: { path: { schoolYearID }, query: { include_deleted: false } },
      }),
    ),
  listInterestAreas: (schoolYearID: string, programID: string, includeRetired = true) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/programs/{programID}/interest-areas", {
        params: { path: { schoolYearID, programID }, query: { include_retired: includeRetired } },
      }),
    ),
  listInterestProfileSurveys: (schoolYearID: string, programID: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys", {
        params: { path: { schoolYearID, programID } },
      }),
    ),
  createInterestProfileSurvey: (
    schoolYearID: string,
    programID: string,
    value: InterestProfileSurveyInput,
  ) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys", {
        params: { path: { schoolYearID, programID } },
        body: value,
      }),
    ),
  getInterestProfileSurvey: (schoolYearID: string, programID: string, surveyID: string) =>
    unwrap(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}",
        { params: { path: { schoolYearID, programID, surveyID } } },
      ),
    ),
  updateInterestProfileSurvey: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    value: InterestProfileSurveyInput,
  ) =>
    unwrap(
      api.PATCH(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}",
        { params: { path: { schoolYearID, programID, surveyID } }, body: value },
      ),
    ),
  deleteInterestProfileSurvey: (schoolYearID: string, programID: string, surveyID: string) =>
    unwrapNoContent(
      api.DELETE(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}",
        { params: { path: { schoolYearID, programID, surveyID } } },
      ),
    ),
  transitionInterestProfileSurvey: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    value: InterestProfileSurveyTransitionInput,
  ) =>
    unwrap(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}/transition",
        { params: { path: { schoolYearID, programID, surveyID } }, body: value },
      ),
    ),
  regenerateInterestProfileSurveyCodes: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    reason: string,
  ) =>
    unwrapList(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}/regenerate-codes",
        { params: { path: { schoolYearID, programID, surveyID } }, body: { reason } },
      ),
    ),
  revokeInterestProfileSurveyCodes: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    reason: string,
  ) =>
    unwrapNoContent(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}/revoke-codes",
        { params: { path: { schoolYearID, programID, surveyID } }, body: { reason } },
      ),
    ),

  getStudentCodeInterestProfileForm: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    organizationID: string,
    code: string,
  ) =>
    unwrap(
      api.POST(
        "/api/respondent/interest-profile-surveys/{schoolYearID}/{programID}/{surveyID}/form",
        {
          params: { path: { schoolYearID, programID, surveyID } },
          body: { organization_id: organizationID, code },
        },
      ),
    ),
  submitStudentCodeInterestProfile: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    organizationID: string,
    code: string,
    answers: PreferenceInterestAnswerInput[],
  ) =>
    unwrap(
      api.POST(
        "/api/respondent/interest-profile-surveys/{schoolYearID}/{programID}/{surveyID}/submit",
        {
          params: { path: { schoolYearID, programID, surveyID } },
          body: { organization_id: organizationID, code, answers: answers ?? [] },
        },
      ),
    ),
  getStudentCodeRankedChoiceForm: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    organizationID: string,
    code: string,
  ) =>
    unwrap(
      api.POST("/api/respondent/sessions/{schoolYearID}/{programID}/{sessionID}/form", {
        params: { path: { schoolYearID, programID, sessionID } },
        body: { organization_id: organizationID, code },
      }),
    ),
  submitStudentCodeRankedChoice: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    organizationID: string,
    code: string,
    responses: PreferenceRankedAnswerInput[],
  ) =>
    unwrap(
      api.POST("/api/respondent/sessions/{schoolYearID}/{programID}/{sessionID}/submit", {
        params: { path: { schoolYearID, programID, sessionID } },
        body: { organization_id: organizationID, code, responses: responses ?? [] },
      }),
    ),
  listGuardianPreferenceForms: () => unwrap(api.GET("/api/guardian/preference-forms")),
  submitGuardianInterestProfile: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    studentID: string,
    answers: PreferenceInterestAnswerInput[],
  ) =>
    unwrap(
      api.POST(
        "/api/guardian/interest-profile-surveys/{schoolYearID}/{programID}/{surveyID}/students/{studentID}",
        {
          params: { path: { schoolYearID, programID, surveyID, studentID } },
          body: { answers: answers ?? [] },
        },
      ),
    ),
  submitGuardianRankedChoice: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    studentID: string,
    responses: PreferenceRankedAnswerInput[],
  ) =>
    unwrap(
      api.POST(
        "/api/guardian/sessions/{schoolYearID}/{programID}/{sessionID}/students/{studentID}",
        {
          params: { path: { schoolYearID, programID, sessionID, studentID } },
          body: { responses: responses ?? [] },
        },
      ),
    ),
  getAdministratorPreferenceForm: (value: Schemas["AdministratorPreferenceFormInputBody"]) =>
    unwrap(api.POST("/api/administrator/preference-form", { body: value })),
  submitAdministratorInterestProfile: (
    schoolYearID: string,
    programID: string,
    surveyID: string,
    studentID: string,
    answers: PreferenceInterestAnswerInput[],
  ) =>
    unwrap(
      api.POST(
        "/api/administrator/interest-profile-surveys/{schoolYearID}/{programID}/{surveyID}/students/{studentID}",
        {
          params: { path: { schoolYearID, programID, surveyID, studentID } },
          body: { answers: answers ?? [] },
        },
      ),
    ),
  submitAdministratorRankedChoice: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    studentID: string,
    responses: PreferenceRankedAnswerInput[],
  ) =>
    unwrap(
      api.POST(
        "/api/administrator/sessions/{schoolYearID}/{programID}/{sessionID}/students/{studentID}",
        {
          params: { path: { schoolYearID, programID, sessionID, studentID } },
          body: { responses: responses ?? [] },
        },
      ),
    ),
  createInterestArea: (schoolYearID: string, programID: string, label: string) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/programs/{programID}/interest-areas", {
        params: { path: { schoolYearID, programID } },
        body: { label },
      }),
    ),
  reorderInterestAreas: (schoolYearID: string, programID: string, ids: string[]) =>
    unwrapList(
      api.POST("/api/school-years/{schoolYearID}/programs/{programID}/interest-areas/reorder", {
        params: { path: { schoolYearID, programID } },
        body: { ids },
      }),
    ),
  updateInterestArea: (
    schoolYearID: string,
    programID: string,
    interestAreaID: string,
    value: Schemas["UpdateInterestAreaInputBody"],
  ) =>
    unwrap(
      api.PATCH(
        "/api/school-years/{schoolYearID}/programs/{programID}/interest-areas/{interestAreaID}",
        { params: { path: { schoolYearID, programID, interestAreaID } }, body: value },
      ),
    ),
  listProgramMemberships: (schoolYearID: string, programID: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/programs/{programID}/memberships", {
        params: { path: { schoolYearID, programID } },
      }),
    ),
  addProgramMembership: (schoolYearID: string, programID: string, studentID: string) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/programs/{programID}/memberships", {
        params: { path: { schoolYearID, programID } },
        body: { student_id: studentID },
      }),
    ),
  removeProgramMembership: (schoolYearID: string, programID: string, membershipID: string) =>
    unwrapNoContent(
      api.DELETE(
        "/api/school-years/{schoolYearID}/programs/{programID}/memberships/{membershipID}",
        { params: { path: { schoolYearID, programID, membershipID } } },
      ),
    ),
  countStudentsWithoutGrade: (schoolYearID: string) =>
    unwrap(
      api.GET("/api/school-years/{schoolYearID}/students/missing-grade-count", {
        params: { path: { schoolYearID } },
      }),
    ),

  listSessions: (schoolYearID: string, programID: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/programs/{programID}/sessions", {
        params: { path: { schoolYearID, programID } },
      }),
    ),
  createSession: (
    schoolYearID: string,
    programID: string,
    value: Schemas["CreateSessionInputBody"],
  ) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/programs/{programID}/sessions", {
        params: { path: { schoolYearID, programID } },
        body: value,
      }),
    ),
  getSession: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrap(
      api.GET("/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}", {
        params: { path: { schoolYearID, programID, sessionID } },
      }),
    ),
  updateSession: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    value: Schemas["UpdateSessionInputBody"],
  ) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}", {
        params: { path: { schoolYearID, programID, sessionID } },
        body: value,
      }),
    ),
  deleteSession: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrapNoContent(
      api.DELETE("/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}", {
        params: { path: { schoolYearID, programID, sessionID } },
      }),
    ),
  transitionSession: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    value: Schemas["TransitionSessionInputBody"],
  ) =>
    unwrap(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/transition",
        { params: { path: { schoolYearID, programID, sessionID } }, body: value },
      ),
    ),
  regenerateRankedChoiceAccessCodes: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    reason: string,
  ) =>
    unwrapList(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/regenerate-codes",
        { params: { path: { schoolYearID, programID, sessionID } }, body: { reason } },
      ),
    ),
  revokeRankedChoiceAccessCodes: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    reason: string,
  ) =>
    unwrapNoContent(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/revoke-codes",
        { params: { path: { schoolYearID, programID, sessionID } }, body: { reason } },
      ),
    ),
  getCatalogFeasibility: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrap(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/catalog-feasibility",
        { params: { path: { schoolYearID, programID, sessionID } } },
      ),
    ),
  listMeetingDates: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrapList(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/meeting-dates",
        { params: { path: { schoolYearID, programID, sessionID } } },
      ),
    ),
  createMeetingDate: (schoolYearID: string, programID: string, sessionID: string, date: string) =>
    unwrap(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/meeting-dates",
        { params: { path: { schoolYearID, programID, sessionID } }, body: { meeting_date: date } },
      ),
    ),
  getMeetingDate: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    meetingDateID: string,
  ) =>
    unwrap(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/meeting-dates/{meetingDateID}",
        { params: { path: { schoolYearID, programID, sessionID, meetingDateID } } },
      ),
    ),
  updateMeetingDate: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    meetingDateID: string,
    date: string,
  ) =>
    unwrap(
      api.PATCH(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/meeting-dates/{meetingDateID}",
        {
          params: { path: { schoolYearID, programID, sessionID, meetingDateID } },
          body: { meeting_date: date },
        },
      ),
    ),
  deleteMeetingDate: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    meetingDateID: string,
  ) =>
    unwrapNoContent(
      api.DELETE(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/meeting-dates/{meetingDateID}",
        { params: { path: { schoolYearID, programID, sessionID, meetingDateID } } },
      ),
    ),
  listOfferings: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrapList(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/offerings",
        { params: { path: { schoolYearID, programID, sessionID } } },
      ),
    ),
  createOffering: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    value: Schemas["CreateOfferingInputBody"],
  ) =>
    unwrap(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/offerings",
        { params: { path: { schoolYearID, programID, sessionID } }, body: value },
      ),
    ),
  getOffering: (schoolYearID: string, programID: string, sessionID: string, offeringID: string) =>
    unwrap(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/offerings/{offeringID}",
        { params: { path: { schoolYearID, programID, sessionID, offeringID } } },
      ),
    ),
  updateOffering: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    offeringID: string,
    value: Schemas["UpdateOfferingInputBody"],
  ) =>
    unwrap(
      api.PATCH(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/offerings/{offeringID}",
        { params: { path: { schoolYearID, programID, sessionID, offeringID } }, body: value },
      ),
    ),
  deleteOffering: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    offeringID: string,
  ) =>
    unwrapNoContent(
      api.DELETE(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/offerings/{offeringID}",
        { params: { path: { schoolYearID, programID, sessionID, offeringID } } },
      ),
    ),
  listSessionNonParticipations: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrapList(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/non-participations",
        { params: { path: { schoolYearID, programID, sessionID } } },
      ),
    ),
  createSessionNonParticipation: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    value: Schemas["CreateSessionNonParticipationInputBody"],
  ) =>
    unwrap(
      api.POST(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/non-participations",
        { params: { path: { schoolYearID, programID, sessionID } }, body: value },
      ),
    ),
  updateSessionNonParticipation: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    nonParticipationID: string,
    value: Schemas["UpdateSessionNonParticipationInputBody"],
  ) =>
    unwrap(
      api.PATCH(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/non-participations/{nonParticipationID}",
        {
          params: { path: { schoolYearID, programID, sessionID, nonParticipationID } },
          body: value,
        },
      ),
    ),
  deleteSessionNonParticipation: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    nonParticipationID: string,
  ) =>
    unwrapNoContent(
      api.DELETE(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/non-participations/{nonParticipationID}",
        { params: { path: { schoolYearID, programID, sessionID, nonParticipationID } } },
      ),
    ),
  getProgramObjectiveWeights: (schoolYearID: string, programID: string) =>
    unwrap(
      api.GET("/api/school-years/{schoolYearID}/programs/{programID}/objective-weights", {
        params: { path: { schoolYearID, programID } },
      }),
    ),
  updateProgramObjectiveWeights: (
    schoolYearID: string,
    programID: string,
    value: ObjectiveWeightsInput,
  ) => {
    const options = { params: { path: { schoolYearID, programID } }, body: value };
    return unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/programs/{programID}/objective-weights", options),
    );
  },
  getSessionObjectiveWeights: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrap(
      api.GET(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/objective-weights",
        { params: { path: { schoolYearID, programID, sessionID } } },
      ),
    ),
  updateSessionObjectiveWeights: (
    schoolYearID: string,
    programID: string,
    sessionID: string,
    value: Schemas["UpdateSessionObjectiveWeightsInputBody"],
  ) => {
    const options = { params: { path: { schoolYearID, programID, sessionID } }, body: value };
    return unwrap(
      api.PATCH(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/objective-weights",
        options,
      ),
    );
  },
  clearSessionObjectiveWeights: (schoolYearID: string, programID: string, sessionID: string) =>
    unwrap(
      api.DELETE(
        "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/objective-weights",
        { params: { path: { schoolYearID, programID, sessionID } } },
      ),
    ),

  getVocabulary: (schoolYearID: string, includeRetired = true) =>
    unwrap(
      api.GET("/api/school-years/{schoolYearID}/vocabularies", {
        params: { path: { schoolYearID }, query: { include_retired: includeRetired } },
      }),
    ),
  updateHomeroomLabel: (homeroom_label: string) =>
    unwrap(api.PATCH("/api/vocabularies/settings", { body: { homeroom_label } })),

  createGradeLevel: (schoolYearID: string, value: Schemas["CreateGradeLevelInputBody"]) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/grade-levels", {
        params: { path: { schoolYearID } },
        body: value,
      }),
    ),
  updateGradeLevel: (
    schoolYearID: string,
    gradeLevelID: string,
    value: Schemas["UpdateGradeLevelInputBody"],
  ) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/grade-levels/{gradeLevelID}", {
        params: { path: { schoolYearID, gradeLevelID } },
        body: value,
      }),
    ),
  reorderGradeLevels: (schoolYearID: string, ids: string[]) =>
    unwrapList(
      api.POST("/api/school-years/{schoolYearID}/grade-levels/reorder", {
        params: { path: { schoolYearID } },
        body: { ids },
      }),
    ),

  createHomeroom: (schoolYearID: string, value: Schemas["CreateHomeroomInputBody"]) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/homerooms", {
        params: { path: { schoolYearID } },
        body: value,
      }),
    ),
  updateHomeroom: (
    schoolYearID: string,
    homeroomID: string,
    value: Schemas["UpdateHomeroomInputBody"],
  ) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/homerooms/{homeroomID}", {
        params: { path: { schoolYearID, homeroomID } },
        body: value,
      }),
    ),

  listAdministrators: () => unwrap(api.GET("/api/administrators")),
  inviteAdministrator: (value: Schemas["InviteAdministratorInputBody"]) =>
    unwrap(api.POST("/api/administrators", { body: value })),
  resendInvitation: (memberID: string) =>
    unwrap(
      api.POST("/api/administrators/{memberID}/invitation/resend", {
        params: { path: { memberID } },
      }),
    ),
  revokeInvitation: (memberID: string) =>
    unwrapNoContent(
      api.POST("/api/administrators/{memberID}/invitation/revoke", {
        params: { path: { memberID } },
      }),
    ),
  changeAdministratorRole: (memberID: string, role: string) =>
    unwrap(
      api.PATCH("/api/administrators/{memberID}", {
        params: { path: { memberID } },
        body: { role },
      }),
    ),
  removeAdministrator: (memberID: string) =>
    unwrapNoContent(
      api.DELETE("/api/administrators/{memberID}", { params: { path: { memberID } } }),
    ),

  previewImport: async (kind: ImportKind, schoolYearID: string, document: File) =>
    unwrap(
      api.POST("/api/imports/{kind}/preview", {
        params: { path: { kind }, query: { school_year_id: schoolYearID } },
        ...(await importBody(kind, document)),
      }),
    ),
  commitImport: async (
    kind: ImportKind,
    schoolYearID: string,
    document: File,
    contentHash: string,
  ) =>
    unwrap(
      api.POST("/api/imports/{kind}/commit", {
        params: {
          path: { kind },
          query: { school_year_id: schoolYearID, content_hash: contentHash },
        },
        ...(await importBody(kind, document)),
      }),
    ),
};

const importContentTypes: Record<ImportKind, string> = {
  roster_json: "application/json",
  grades_csv: "text/csv",
};

// An import endpoint takes the organiser's file byte for byte: the content
// hash in the preview must cover the reviewed document, and the registered
// parser is the only thing entitled to interpret it. The client's default
// serializer would JSON.stringify the text, which turns a roster array into a
// JSON string literal and a CSV into a quoted blob, so it is replaced here
// rather than worked around in the caller.
async function importBody(kind: ImportKind, document: File) {
  return {
    body: await document.text(),
    // The parameter type is the union of the operation's declared content
    // types; only the string arm is ever passed, and it is passed through.
    bodySerializer: (body: string | Record<string, never>[]) => body as string,
    headers: { "Content-Type": importContentTypes[kind] },
  };
}

export function activeGradeLevels(vocabulary: VocabularyResponse): GradeLevel[] {
  return (vocabulary.grade_levels ?? [])
    .filter((grade) => !grade.retired_at)
    .sort((a, b) => a.ordinal - b.ordinal);
}

export function activeHomerooms(vocabulary: VocabularyResponse): Homeroom[] {
  return (vocabulary.homerooms ?? []).filter((homeroom) => !homeroom.retired_at);
}
