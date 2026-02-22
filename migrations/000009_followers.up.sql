CREATE TABLE IF NOT EXISTS followers(
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    follower_id UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY(user_id, follower_id)
);