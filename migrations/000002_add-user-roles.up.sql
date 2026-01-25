CREATE TYPE role_type AS ENUM ('READER', 'WRITER');
ALTER TABLE users ADD COLUMN roles role_type[] NOT NULL DEFAULT ARRAY['READER']::role_type[];