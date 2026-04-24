CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    company_id UUID,
    created_at TIMESTAMP NULL DEFAULT now()
);