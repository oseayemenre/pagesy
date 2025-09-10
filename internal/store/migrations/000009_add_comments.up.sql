CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- Comment section which is everything below, would be used in the following tables
       -- on its own, for reports, forums and book clubs

CREATE TABLE IF NOT EXISTS comments(
    id TEXT NOT NULL UNIQUE,
    userid UUID REFERENCES users(id)
    content NOT NULL TEXT,
    isauthor bit,
    entity_id TEXT,
    vote_id UUID REFERENCES votes(id),
    image TEXT,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
)

CREATE TABLE IF NOT EXISTS votes(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    commentid UUID REFERENCES comments(id),
    vote NOT NULL bit,
    userid UUID NOT NULL UNIQUE REFERENCES users(id),
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
)

