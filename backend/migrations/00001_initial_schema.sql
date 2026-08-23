-- +goose Up

CREATE TABLE health_checks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    status TEXT NOT NULL DEFAULT 'healthy',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT health_checks_status_check CHECK (status <> '')
);

-- +goose Down

DROP TABLE IF EXISTS health_checks;
