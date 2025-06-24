CREATE TYPE genre_type AS ENUM(
  'Romance',
  'Action',
  'Mystery',
  'Thriller',
  'Science Fiction',
  'Fantasy',
  'Horror',
  'Historical',
  'Biography/Memoir',
  'Children',
  'Young Adult',
  'Poetry'
);

ALTER TABLE genres
ALTER COLUMN genres TYPE genre_type
USING genres::genre_type;

