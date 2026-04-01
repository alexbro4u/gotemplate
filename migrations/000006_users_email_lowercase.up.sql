UPDATE users SET email = LOWER(email);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'chk_users_email_lowercase'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT chk_users_email_lowercase CHECK (email = LOWER(email));
    END IF;
END $$;
