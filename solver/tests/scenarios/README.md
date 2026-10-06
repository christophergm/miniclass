# Synthetic solver scenarios

Each directory is a standalone normalized CSV scenario. The harness can also
combine its input directories before compiling a self-contained v1 solve
request, allowing rule interactions without a database or a production export.

Required inputs are `students.csv` (`id,grade_ordinal`), `offerings.csv`
(`id,capacity,min_grade_ordinal,max_grade_ordinal,interest_area_id`),
`interest-ratings.csv` (`participant_id,interest_area_id,rating`), and
`ranked-choices.csv` (`participant_id,offering_id,response,rank`). Optional
`pins.csv`, `exclusions.csv`, and `prior-placements.csv` each have
`participant_id,offering_id`; `authorized-pinned-exceptions.csv` has
`participant_id,offering_id,rule`. Empty optional fields are encoded as empty
CSV cells. `expected.csv` has
`status,participant_id,offering_id,realized_quality`; an assignment may be
omitted when the scenario asserts only a status and general solver invariants.

All identifiers are opaque synthetic test tokens. Deliberately no display names
or roster attributes are permitted here.
