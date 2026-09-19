-- ==========================================
-- 1. MODULE USER & AUTHENTICATION
-- ==========================================
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    full_name TEXT NOT NULL,
    phone_number TEXT,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'customer', -- 'customer' hoặc 'admin'
    status TEXT NOT NULL DEFAULT 'active',   -- 'active' hoặc 'inactive'
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS blacklisted_tokens (
    id TEXT PRIMARY KEY,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

-- ==========================================
-- 2. MODULE PRODUCT (NÔNG SẢN TƯƠI SỐNG)
-- ==========================================
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price REAL NOT NULL,
    unit TEXT NOT NULL,                  -- Đơn vị tính: 'mớ', 'kg', 'túi 500g', 'con'
    stock_quantity INTEGER NOT NULL DEFAULT 0, -- Sản lượng thực tế thu hoạch ở vườn
    category TEXT,                       -- 'rau_an_la', 'cu_qua', 'thuy_san'
    image_url TEXT,
    status TEXT DEFAULT 'active',        -- 'active' (đang bán), 'out_of_season' (hết mùa)
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME DEFAULT NULL     -- Xóa mềm, bảo toàn lịch sử đơn hàng
);

-- ==========================================
-- 3. MODULE ORDER (ĐƠN HÀNG & GIAO NHẬN)
-- ==========================================
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    total_amount REAL NOT NULL,
    status TEXT NOT NULL,                -- 'pending', 'confirmed', 'shipping', 'completed', 'cancelled'
    order_type TEXT NOT NULL,            -- 'pickup' (Lấy tại vườn), 'delivery' (Giao tận nơi)
    payment_method TEXT NOT NULL,        -- 'cod', 'banking', 'momo'
    shipping_address TEXT,               -- Cho phép NULL nếu khách chọn pickup tại vườn
    delivery_date TEXT,                  -- Ngày gom đơn giao cụ thể (VD: "2026-09-21")
    note TEXT,                           -- Ghi chú của khách (VD: "Làm sạch cá giúp mình")
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price REAL NOT NULL,            -- Lưu giá tại thời điểm chốt đơn
    sub_total REAL NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- ==========================================
-- 4. MODULE MANAGEMENT & LOGGING (VẬN HÀNH)
-- ==========================================
-- Lịch sử thay đổi trạng thái đơn hàng (Truy vết minh bạch cho chủ vườn và khách)
CREATE TABLE IF NOT EXISTS order_status_history (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    old_status TEXT NOT NULL,
    new_status TEXT NOT NULL,
    changed_by TEXT NOT NULL,            -- ID của User hoặc Admin thực hiện chuyển trạng thái
    note TEXT,                           -- Lý do đổi trạng thái (nếu có)
    created_at DATETIME NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Quản lý thanh toán và đối soát dòng tiền
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    amount REAL NOT NULL,
    provider TEXT NOT NULL,              -- 'cod', 'banking', 'momo'
    transaction_id TEXT,                 -- Mã giao dịch từ cổng thanh toán (nếu có)
    status TEXT NOT NULL,                -- 'pending', 'success', 'failed', 'refunded'
    paid_at DATETIME,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);