CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    phone_number TEXT,
    avatar_url TEXT,
    role TEXT DEFAULT 'customer',
    status TEXT DEFAULT 'active',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME DEFAULT NULL
);

-- Lưu trữ Refresh Token để cấp lại Access Token
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Lưu trữ JTI (ID duy nhất của Access Token) khi người dùng Đăng xuất
CREATE TABLE IF NOT EXISTS blacklisted_tokens (
    jti TEXT PRIMARY KEY,
    expires_at DATETIME NOT NULL
);