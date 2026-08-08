-- Singleton table: "id boolean primary key default true check (id)" makes a
-- second row physically impossible to insert, so callers can always assume
-- exactly one row exists (seeded below) without a WHERE clause.
CREATE TABLE app_settings (
    id                       BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),

    storage_backend          TEXT NOT NULL DEFAULT 'local' CHECK (storage_backend IN ('local', 's3', 'gcs')),

    -- Relative so it resolves sensibly both in the Docker image (working
    -- dir "/", so this becomes "/data", matching the mounted volume) and
    -- for a bare `go run`/`make run` outside Docker.
    local_dir                TEXT NOT NULL DEFAULT './data',

    s3_bucket                TEXT NOT NULL DEFAULT '',
    s3_region                TEXT NOT NULL DEFAULT '',
    s3_prefix                TEXT NOT NULL DEFAULT '',
    s3_endpoint              TEXT NOT NULL DEFAULT '',
    s3_use_path_style        BOOLEAN NOT NULL DEFAULT false,
    -- Encrypted (AES-256-GCM) with the server's SETTINGS_ENCRYPTION_KEY, not
    -- plaintext. NULL means "no static credentials configured" (falls back
    -- to the AWS default credential chain / IAM role).
    s3_access_key_id_enc     TEXT,
    s3_secret_access_key_enc TEXT,

    gcs_bucket               TEXT NOT NULL DEFAULT '',
    gcs_prefix               TEXT NOT NULL DEFAULT '',
    -- Encrypted service-account JSON key. NULL falls back to Application
    -- Default Credentials (workload identity, gcloud ADC, etc).
    gcs_credentials_json_enc TEXT,

    session_ttl_seconds      INT NOT NULL DEFAULT 86400,
    max_cache_entry_bytes    BIGINT NOT NULL DEFAULT 524288000,

    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by               BIGINT REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO app_settings (id) VALUES (true);
