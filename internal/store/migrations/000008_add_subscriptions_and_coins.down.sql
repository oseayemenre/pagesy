ALTER TABLE users DROP COLUMN coins;
ALTER TABLE books DROP COLUMN subscription;
DROP TRIGGER IF EXISTS update_updated_at_on_renewal ON subscriptions;
DROP FUNCTION IF EXISTS update_updated_at_on_renewal_func;
DROP TABLE IF EXISTS subscriptions;
DROP TYPE IF EXISTS plan_type;
DROP TYPE IF EXISTS status_type;
