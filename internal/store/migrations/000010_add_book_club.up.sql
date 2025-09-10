CREATE TABLE IF NOT EXISTS book_club(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    bk_name TEXT NOT NULL UNIQUE,
    image TEXT,
    creator_id UUID REFERENCES users(id),
    description TEXT NOT NULL,
    isopen bit,
    modified_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
        );

CREATE TABLE IF NOT EXIST bk_moderator(
    book_id UUID REFERENCES book_club(id),
    user_id UUID REFERENCES users(id),
    Abilities TEXT,
    modified_by UUID REFERENCES users(id),
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP

);