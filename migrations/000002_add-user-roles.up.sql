CREATE TYPE role_type AS ENUM ('REGULAR', 'ADMIN');
ALTER TABLE users ADD COLUMN roles role_type[] NOT NULL DEFAULT ARRAY['REGULAR']::role_type[];