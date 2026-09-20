-- +goose Up

grant select, delete on interest_profile_submissions,
    ranked_choice_submissions,
    interest_profile_survey_audience_students,
    interest_profile_survey_audience_snapshots,
    interest_profile_survey_access_codes,
    ranked_choice_access_codes to miniclass_app;

-- +goose Down

revoke delete on interest_profile_submissions,
    ranked_choice_submissions,
    interest_profile_survey_audience_students,
    interest_profile_survey_audience_snapshots,
    interest_profile_survey_access_codes,
    ranked_choice_access_codes from miniclass_app;
