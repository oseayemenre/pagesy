CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TYPE day_type AS ENUM(
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday'
);
CREATE TYPE language_type AS ENUM(
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

CREATE TABLE IF NOT EXISTS books(
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT UNIQUE NOT NULL,
  description TEXT NOT NULL,
  image TEXT,
  views INT NOT NULL DEFAULT 0,
  language language_type NOT NULL DEFAULT 'English'::language_type,
  rating INT NOT NULL DEFAULT 0,
  author_id UUID REFERENCES users(id) ON DELETE CASCADE,
  completed BOOLEAN NOT NULL DEFAULT FALSE,
  approved BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS release_schedule(
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  day day_type NOT NULL,
  no_of_chapters INT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chapters(
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  chapter_no INT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS genres(
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  genre genre_type NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO genres(genre)
VALUES
  ('Romance'),
  ('Action'),
  ('Mystery'),
  ('Thriller'),
  ('Science Fiction'),
  ('Fantasy'),
  ('Horror'),
  ('Historical'),
  ('Biography/Memoir'),
  ('Children'),
  ('Young Adult'),
  ('Poetry');


CREATE TABLE IF NOT EXISTS books_genres(
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  genre_id UUID NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
  PRIMARY KEY(book_id, genre_id)
);

CREATE INDEX IF NOT EXISTS idx_release_schedule_book_id ON release_schedule(book_id);
CREATE INDEX IF NOT EXISTS idx_chapters_book_id ON chapters(book_id);