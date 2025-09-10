-- Report section which is everything below utilized in reporting system errors or user mischief

CREATE TYPE report_category AS ENUM(
    'System',
    'User'
);
CREATE TYPE report_type AS ENUM(
    'Sexual Content',
    'Violent Content',
    'Harassment',
    'Spam',
    'Others'
);

CREATE TABLE IF NOT EXISTS report(
    id TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id),
    category report_category NOT NULL,
    report report_type,
    comment_id UUID REFERENCES comments(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);