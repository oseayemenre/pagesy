-- Comment section which is everything below, would be used in the following tables
       -- on its own, for reports, forums and book clubs
CREATE TYPE comment_category AS ENUM(
    'Comments',
    'Reports',
    'Review',
    'Posts',
    'Forum'
);
CREATE TYPE entity_category AS ENUM(
    'Comments',
    'Books',
    'Posts'
);
CREATE TABLE IF NOT EXISTS comments(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    category comment_category NOT NULL,
    user_id UUID REFERENCES users(id),
    content TEXT NOT NULL,
    isauthor bit,
    isexclusive bit,
    ispost bit,
    pinned bit,
    entity_id UUID,
    entity_type entity_category,
    vote_id UUID REFERENCES votes(id),
    image TEXT,
    seen bit,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS votes(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID REFERENCES comments(id),
    vote NOT NULL int, -- int because this also used for polls
    user_id UUID NOT NULL UNIQUE REFERENCES users(id),
    seen bit,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
);


