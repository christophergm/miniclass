-- name: CreateProgramObjectiveWeights :one
insert into program_objective_weights (organization_id, school_year_id, program_id)
values ($1, $2, $3)
returning id, organization_id, school_year_id, program_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at;

-- name: GetProgramObjectiveWeights :one
select id, organization_id, school_year_id, program_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at
from program_objective_weights
where organization_id = $1 and school_year_id = $2 and program_id = $3;

-- name: UpdateProgramObjectiveWeights :one
update program_objective_weights
set rank_high_max = $4, deficit_unwanted_increment = $5, deficit_neutral_increment = $6,
    deficit_acceptable_increment = $7, deficit_influence = $8,
    repeat_offering_penalty = $9, repeat_interest_area_penalty = $10,
    tag_prefers_weight = $11, tag_discourages_weight = $12,
    pairing_prefers_weight = $13, pairing_discourages_weight = $14,
    below_minimum_enrollment_penalty = $15, tag_balance_penalty = $16
where organization_id = $1 and school_year_id = $2 and program_id = $3
returning id, organization_id, school_year_id, program_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at;

-- name: GetSessionObjectiveWeightOverrides :one
select id, organization_id, school_year_id, program_id, session_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at
from session_objective_weight_overrides
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4;

-- name: UpsertSessionObjectiveWeightOverrides :one
insert into session_objective_weight_overrides (
    organization_id, school_year_id, program_id, session_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
on conflict (organization_id, school_year_id, program_id, session_id) do update set
    rank_high_max = excluded.rank_high_max,
    deficit_unwanted_increment = excluded.deficit_unwanted_increment,
    deficit_neutral_increment = excluded.deficit_neutral_increment,
    deficit_acceptable_increment = excluded.deficit_acceptable_increment,
    deficit_influence = excluded.deficit_influence,
    repeat_offering_penalty = excluded.repeat_offering_penalty,
    repeat_interest_area_penalty = excluded.repeat_interest_area_penalty,
    tag_prefers_weight = excluded.tag_prefers_weight,
    tag_discourages_weight = excluded.tag_discourages_weight,
    pairing_prefers_weight = excluded.pairing_prefers_weight,
    pairing_discourages_weight = excluded.pairing_discourages_weight,
    below_minimum_enrollment_penalty = excluded.below_minimum_enrollment_penalty,
    tag_balance_penalty = excluded.tag_balance_penalty
returning id, organization_id, school_year_id, program_id, session_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at;

-- name: DeleteSessionObjectiveWeightOverrides :execrows
delete from session_objective_weight_overrides
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4;

-- name: ListAllProgramObjectiveWeightsForRegistry :many
select id, organization_id, school_year_id, program_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at
from program_objective_weights where organization_id = $1 order by id;

-- name: FindProgramObjectiveWeightsForRegistry :one
select id, organization_id, school_year_id, program_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at
from program_objective_weights where id = $1 and organization_id = $2;

-- name: UpdateProgramObjectiveWeightsForRegistry :execrows
update program_objective_weights set updated_at = now() where id = $1 and organization_id = $2;

-- name: DeleteProgramObjectiveWeightsForRegistry :execrows
delete from program_objective_weights where id = $1 and organization_id = $2;

-- name: ListAllSessionObjectiveWeightOverridesForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at
from session_objective_weight_overrides where organization_id = $1 order by id;

-- name: FindSessionObjectiveWeightOverridesForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, rank_high_max,
    deficit_unwanted_increment, deficit_neutral_increment, deficit_acceptable_increment,
    deficit_influence, repeat_offering_penalty, repeat_interest_area_penalty,
    tag_prefers_weight, tag_discourages_weight, pairing_prefers_weight,
    pairing_discourages_weight, below_minimum_enrollment_penalty, tag_balance_penalty,
    created_at, updated_at
from session_objective_weight_overrides where id = $1 and organization_id = $2;

-- name: UpdateSessionObjectiveWeightOverridesForRegistry :execrows
update session_objective_weight_overrides set updated_at = now() where id = $1 and organization_id = $2;

-- name: DeleteSessionObjectiveWeightOverridesForRegistry :execrows
delete from session_objective_weight_overrides where id = $1 and organization_id = $2;
