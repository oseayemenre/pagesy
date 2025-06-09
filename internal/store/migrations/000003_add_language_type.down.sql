DROP TYPE IF EXISTS languages;
ALTER TABLE books
ALTER COLUMN language TYPE text USING language::text;
