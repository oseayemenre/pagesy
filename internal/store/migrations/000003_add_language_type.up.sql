CREATE TYPE languages AS ENUM(
  'Mandarin Chinese',
  'Spanish',
  'English',
  'Hindi',
  'Portugese',
  'Vietnamese',
  'Russian',
  'Japanese',
  'Korean',
  'Indonesian'
);

ALTER TABLE books
ALTER COLUMN language DROP DEFAULT,
ALTER COLUMN language TYPE languages USING language::languages, 
ALTER COLUMN language SET DEFAULT 'English'; 
