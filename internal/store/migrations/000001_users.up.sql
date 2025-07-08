CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users(
  id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  username TEXT,
  email TEXT NOT NULL UNIQUE,
  password TEXT,
  name TEXT,
  image TEXT,
  about TEXT,
  followers UUID REFERENCES users(id),
  following UUID REFERENCES users(id),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_user_table_updated_at()
RETURNS TRIGGER
AS $$
BEGIN
  NEW.updated_at := NOW();
  RETURN NEW;
END $$ language plpgsql;

CREATE TRIGGER update_user_table_updated_at_trigger
BEFORE UPDATE on users
FOR EACH ROW
  EXECUTE FUNCTION update_user_table_updated_at();
