CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- Comment section which is everything below, would be used in the following tables
       -- on its own, for reports, forums and book clubs
CREATE TYPE comment_category AS ENUM(
    'Comments',
    'Reports',
    'Review'
    'Forums',
    'Posts'
);
CREATE TYPE entity_category AS ENUM(
    'Comments',
    'Reports',
);
CREATE TABLE IF NOT EXISTS comments(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    category comment_category NOT NULL,
    user_id UUID REFERENCES users(id),
    content TEXT NOT NULL,
    isauthor bit,
    entity_id UUID,
    entity_type entity_category,
    vote_id UUID REFERENCES votes(id),
    image TEXT,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS votes(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID REFERENCES comments(id),
    vote NOT NULL bit,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id),
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

