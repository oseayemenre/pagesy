CREATE TYPE role_type AS ENUM ('READER', 'WRITER', 'ADMIN');
ALTER TABLE users ADD COLUMN roles role_type[] NOT NULL DEFAULT ARRAY['READER']::role_type[];