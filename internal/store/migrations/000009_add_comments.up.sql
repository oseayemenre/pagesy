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
    'Chapters',
    'Posts'
);

CREATE TABLE IF NOT EXISTS votes(
                                    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID, -- We'll add the foreign key later
    vote int NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    seen bool DEFAULT false,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
                              );

CREATE TABLE IF NOT EXISTS comments(
                                       id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    category comment_category NOT NULL,
    user_id UUID REFERENCES users(id) NOT NULL,
    content TEXT NOT NULL CHECK(LENGTH(content)>0),
    is_author bool NOT NULL DEFAULT false,
    is_exclusive bool NOT NULL DEFAULT false,
    is_post bool NOT NULL DEFAULT false,
    pinned bool NOT NULL DEFAULT false,
    entity_id UUID,
    entity_type entity_category,
    vote_id UUID REFERENCES votes(id),
    image TEXT,
    seen bool NOT NULL DEFAULT false,
    is_deleted bool NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
                              );

ALTER TABLE votes ADD CONSTRAINT fk_votes_comment
    FOREIGN KEY (comment_id) REFERENCES comments(id);

CREATE INDEX idx_comments_category ON comments(category);
CREATE INDEX idx_comments_entity ON comments(entity_type, entity_id);
CREATE INDEX idx_comments_user ON comments(user_id);
CREATE INDEX idx_comments_active ON comments(is_deleted, modified_at DESC) WHERE is_deleted = false;
CREATE INDEX idx_votes_comment ON votes(comment_id);

ALTER TABLE comments ADD CONSTRAINT entity_check
    CHECK ((entity_id IS NULL AND entity_type IS NULL) OR (entity_id IS NOT NULL AND entity_type IS NOT NULL));