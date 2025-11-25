-- Create users table for authentication
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    role VARCHAR(50) DEFAULT 'user',
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active);

-- Create default admin user (password: admin123)
-- bcrypt hash of 'admin123' with cost 10
INSERT INTO users (username, email, password_hash, role, is_active)
VALUES (
    'admin',
    'admin@trufflehog.local',
    '$2a$10$aK5XHp5zZ6vWAHWq3m1sO.aUbZSaBqLW295zMDJ1YuYR0aDjdwVQa',
    'admin',
    TRUE
) ON CONFLICT (username) DO NOTHING;

-- Update trigger for users
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

