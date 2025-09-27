CREATE TABLE IF NOT EXISTS reviews(
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    rating UUID float,
    book_id UUID NOT NULL UNIQUE REFERENCES books(id),
    comment_id UUID REFERENCES comments(id),
    seen bit,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                             );