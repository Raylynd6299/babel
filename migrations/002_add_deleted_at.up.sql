-- Add deleted_at column to users table
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL;

-- Add index for deleted_at for better performance
CREATE INDEX idx_users_deleted_at ON users(deleted_at);