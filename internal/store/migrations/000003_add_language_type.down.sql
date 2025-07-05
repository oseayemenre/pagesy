ALTER TABLE books
ALTER COLUMN language DROP DEFAULT;

ALTER TABLE books
ALTER COLUMN language TYPE text USING language::text;

DROP TYPE IF EXISTS languages;
