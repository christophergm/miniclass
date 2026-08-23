-- +goose Up

-- PostgreSQL 16 does not provide uuidv7(), so keep generation local to the
-- schema while preserving the UUID v7 layout for every generated ID.
CREATE FUNCTION miniclass_uuid_v7() RETURNS UUID
LANGUAGE SQL
VOLATILE
AS $$
    WITH random_value AS (
        SELECT md5(random()::TEXT || clock_timestamp()::TEXT) AS hex
    ), timestamp_value AS (
        SELECT lpad(
            to_hex(floor(extract(EPOCH FROM clock_timestamp()) * 1000)::BIGINT),
            12,
            '0'
        ) AS hex
    )
    SELECT (
        timestamp_value.hex
        || '7'
        || substr(random_value.hex, 14, 3)
        || substr(
            '89ab',
            (get_byte(decode(substr(random_value.hex, 17, 2), 'hex'), 0) % 4) + 1,
            1
        )
        || substr(random_value.hex, 18)
    )::UUID
    FROM random_value, timestamp_value;
$$;

CREATE TABLE health_checks (
    id UUID PRIMARY KEY DEFAULT miniclass_uuid_v7(),
    status TEXT NOT NULL DEFAULT 'healthy',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT health_checks_status_check CHECK (status <> '')
);

-- +goose Down

DROP TABLE IF EXISTS health_checks;
DROP FUNCTION IF EXISTS miniclass_uuid_v7();
