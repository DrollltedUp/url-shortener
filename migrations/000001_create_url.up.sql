CREATE TABLE url (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url_original TEXT NOT NULL,
    url_short    TEXT NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);