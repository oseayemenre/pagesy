ALTER TABLE users ADD CONSTRAINT password_length CHECK (char_length(password) > 7);
