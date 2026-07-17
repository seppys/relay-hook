ALTER TABLE events ADD COLUMN user_id UUID REFERENCES users(id);
ALTER TABLE subscribers ADD COLUMN user_id UUID REFERENCES users(id);