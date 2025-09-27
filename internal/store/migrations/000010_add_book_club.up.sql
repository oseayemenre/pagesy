CREATE TYPE comment_category AS ENUM(
    'Comments',
    'Reports',
    'Review',
    'Posts',
    'Forum'
);
CREATE TABLE IF NOT EXISTS book_club(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    clb_name TEXT NOT NULL UNIQUE,
    image TEXT,
    creator_id UUID REFERENCES users(id),
    description TEXT NOT NULL,
    isopen bool,
    modified_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
        );

CREATE TABLE IF NOT EXIST bk_moderator(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    bookclb_id UUID REFERENCES book_club(id),
    user_id UUID REFERENCES users(id),
    Abilities TEXT,
    modified_by UUID REFERENCES users(id),
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP

);