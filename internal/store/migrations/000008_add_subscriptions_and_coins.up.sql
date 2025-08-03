ALTER TABLE users
ADD COLUMN coins INT NOT NULL DEFAULT 0 
CHECK(coins >= 0);

ALTER TABLE books
ADD COLUMN subscription BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TYPE status_type AS ENUM('active', 'inactive');

CREATE TYPE plan_type AS ENUM('basic', 'premium');

CREATE TABLE IF NOT EXISTS subscriptions (
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plan plan_type NOT NULL,
  status status_type NOT NULL DEFAULT 'active',
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_updated_at_on_renewal_func()
RETURNS TRIGGER
AS $$
  BEGIN
    UPDATE subscriptions
    SET updated_at = NOW()
    WHERE book_id = OLD.book_id AND user_id = OLD.user_id;

    RETURN NEW;
  END; $$ language 'plpgsql';

CREATE TRIGGER update_updated_at_on_renewal
BEFORE UPDATE ON subscriptions
FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_on_renewal_func();


