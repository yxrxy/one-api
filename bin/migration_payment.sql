-- 支付记录表
CREATE TABLE IF NOT EXISTS payment_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    order_id VARCHAR(100) UNIQUE NOT NULL,
    top_up_code VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 充值码折扣表
CREATE TABLE IF NOT EXISTS top_up_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code VARCHAR(100) UNIQUE NOT NULL,
    discount DECIMAL(3,2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'enabled',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_payment_records_user_id ON payment_records(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_records_status ON payment_records(status);
CREATE INDEX IF NOT EXISTS idx_payment_records_created_at ON payment_records(created_at);
CREATE INDEX IF NOT EXISTS idx_top_up_codes_status ON top_up_codes(status);

-- 插入一些示例充值码
INSERT OR IGNORE INTO top_up_codes (code, discount, status) VALUES 
('WELCOME10', 0.10, 'enabled'),
('NEWUSER20', 0.20, 'enabled'),
('VIP15', 0.15, 'enabled'); 