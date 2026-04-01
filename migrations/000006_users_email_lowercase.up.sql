UPDATE users SET email = LOWER(email);
ALTER TABLE users ADD CONSTRAINT chk_users_email_lowercase CHECK (email = LOWER(email));
