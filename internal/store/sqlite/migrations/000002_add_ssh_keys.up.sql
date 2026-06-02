-- Add SSH keys table for key-based authentication

CREATE TABLE ssh_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    public_key TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_ssh_keys_user_id ON ssh_keys(user_id);
CREATE INDEX idx_ssh_keys_fingerprint ON ssh_keys(fingerprint);

-- Ensure unique fingerprints to prevent duplicate keys
CREATE UNIQUE INDEX idx_ssh_keys_fingerprint_unique ON ssh_keys(fingerprint);
