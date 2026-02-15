## 代码结构 
```text
ForecastSync/
├── cmd/
│   └── server/
│       └── main.go          # 入口文件（启动API服务）
├── internal/
│   ├── adapter/             # 平台适配器层（每个平台一个目录）
│   │   ├── polymarket/      # Polymarket适配器
│   │   │   ├── adapter.go   # Polymarket适配器实现
│   │   └── kalshi/          # Kalshi适配器
│   │       ├── adapter.go   # Kalshi适配器实现
│   ├── config/              # 配置管理
│   │   └── config.go        # 全局配置
│   ├── interfaces/          # 通用接口定义
│   │   └── platform_adapter.go  # 平台适配器通用接口
│   ├── model/               # 数据库模型 + 通用数据结构
│   │   ├── db.go            # 数据库表模型（Event/EventOdds等）
│   │   └── platform.go      # 通用平台数据结构
│   ├── repository/          # 通用数据库操作
│   │   └── event_repo.go    # Event/EventOdds入库逻辑
│   ├── service/             # 通用同步服务
│   │   └── sync.go          # 多平台同步核心逻辑
│   └── api/                 # API接口层
│       └── sync_handler.go  # 同步接口
│   └── utils/               # API接口层
│       └── httpclient       # 统一封装http请求工具类
│   │       ├── adapter.go   # http工具类实现
├── go.mod
└── go.sum
```

## 库表结构

说明：应用启动时会**自动创建不存在的数据库**（需能连上 `postgres` 默认库），并执行 GORM 迁移**表不存在则按 model 自动创建**。若需手动初始化或与 Go 模型保持一致，可在 `forecast_aggregation` 库中执行下方完整 SQL（幂等，可重复执行）。

```sql
-- 创建数据库（仅首次或单独执行）
-- CREATE DATABASE forecast_aggregation
--   WITH OWNER = postgres ENCODING = 'UTF8' LC_COLLATE = 'en_US.UTF-8' LC_CTYPE = 'en_US.UTF-8'
--   TABLESPACE = pg_default CONNECTION LIMIT = -1;

SET TIME ZONE 'UTC';

-- 前提：已创建 forecast_aggregation 并连接到该库

-- ------------------------------
-- 1. 用户表（users）
-- ------------------------------
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    wallet_address VARCHAR(64) NOT NULL UNIQUE,
    total_profit NUMERIC(18,6) DEFAULT 0,
    total_loss NUMERIC(18,6) DEFAULT 0,
    total_fee NUMERIC(18,6) DEFAULT 0,
    gas_fee_total NUMERIC(18,6) DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
COMMENT ON TABLE users IS '用户基础信息表，关联钱包地址作为唯一标识';
COMMENT ON COLUMN users.id IS '自增主键ID';
COMMENT ON COLUMN users.wallet_address IS '用户钱包地址（0x开头，小写存储）';
COMMENT ON COLUMN users.total_profit IS '用户累计盈利（USDC，保留6位小数）';
COMMENT ON COLUMN users.total_loss IS '用户累计亏损（USDC，保留6位小数）';
COMMENT ON COLUMN users.total_fee IS '用户累计支付的1%平台管理费（USDC）';
COMMENT ON COLUMN users.gas_fee_total IS '用户累计支付的链上Gas费（换算为USDC）';
COMMENT ON COLUMN users.is_active IS '用户是否活跃：true=活跃，false=禁用';
COMMENT ON COLUMN users.created_at IS '用户创建时间（首次登录时间）';
COMMENT ON COLUMN users.updated_at IS '用户信息更新时间';
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
COMMENT ON INDEX idx_users_created_at IS '用户创建时间索引，用于按时间筛选用户';

-- ------------------------------
-- 2. 第三方平台配置表（platforms）
-- ------------------------------
CREATE TABLE IF NOT EXISTS platforms (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(32) NOT NULL,
    type VARCHAR(16) NOT NULL,
    api_url VARCHAR(256),
    contract_address VARCHAR(64),
    rpc_url VARCHAR(256),
    api_key VARCHAR(128),
    api_limit INT DEFAULT 600,
    current_api_usage INT DEFAULT 0,
    is_hot BOOLEAN DEFAULT FALSE,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
COMMENT ON TABLE platforms IS '第三方预测平台配置表，管理多平台对接参数';
COMMENT ON COLUMN platforms.id IS '自增主键ID';
COMMENT ON COLUMN platforms.name IS '第三方预测平台名称（如Polymarket、Kalshi）';
COMMENT ON COLUMN platforms.type IS '平台类型：chain=链上平台，centralized=中心化平台';
COMMENT ON COLUMN platforms.api_url IS '中心化平台API接口地址';
COMMENT ON COLUMN platforms.contract_address IS '链上平台核心合约地址（0x开头）';
COMMENT ON COLUMN platforms.rpc_url IS '链上平台RPC节点地址（如Polygon RPC）';
COMMENT ON COLUMN platforms.api_key IS 'API密钥（AES加密存储）';
COMMENT ON COLUMN platforms.api_limit IS '单API Key每分钟调用限额';
COMMENT ON COLUMN platforms.current_api_usage IS '当前分钟已调用API次数（实时更新）';
COMMENT ON COLUMN platforms.is_hot IS '是否为热门平台：true=是，false=否（优先缓存）';
COMMENT ON COLUMN platforms.is_enabled IS '平台是否启用：true=启用，false=禁用';
COMMENT ON COLUMN platforms.created_at IS '平台配置创建时间';
COMMENT ON COLUMN platforms.updated_at IS '平台配置更新时间';
CREATE INDEX IF NOT EXISTS idx_platforms_name ON platforms(name);
CREATE INDEX IF NOT EXISTS idx_platforms_type ON platforms(type);
CREATE INDEX IF NOT EXISTS idx_platforms_is_hot ON platforms(is_hot);
CREATE INDEX IF NOT EXISTS idx_platforms_is_enabled ON platforms(is_enabled);
-- 初始化平台数据（存在则跳过）
INSERT INTO platforms (id, name, type, api_url, contract_address, rpc_url, api_key, api_limit, current_api_usage, is_hot, is_enabled, created_at, updated_at)
VALUES (1, 'polymarket', 'centralized', 'https://gamma-api.polymarket.com', NULL, NULL, NULL, 600, 0, FALSE, TRUE, NOW(), NOW()),
       (2, 'kalshi', 'centralized', 'https://api.kalshi.com/v1', NULL, NULL, NULL, 600, 0, FALSE, TRUE, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ------------------------------
-- 3. 预测事件主表（events）
-- ------------------------------
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_uuid VARCHAR(128) NOT NULL UNIQUE,
    title VARCHAR(256) NOT NULL,
    type VARCHAR(16) NOT NULL,
    platform_id BIGINT NOT NULL REFERENCES platforms(id),
    platform_event_id VARCHAR(128) NOT NULL,
    canonical_key VARCHAR(64),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    resolve_time TIMESTAMP,
    options JSONB NOT NULL,
    result VARCHAR(32),
    result_source VARCHAR(256),
    result_verified BOOLEAN DEFAULT FALSE,
    status VARCHAR(16) DEFAULT 'active',
    is_hot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_events_platform_event UNIQUE (platform_id, platform_event_id)
);
COMMENT ON TABLE events IS '预测事件主表，存储所有对接平台的预测事件信息';
COMMENT ON COLUMN events.id IS '自增主键ID';
COMMENT ON COLUMN events.event_uuid IS '全局唯一事件ID，规则：platform_id_platform_event_id（确定性）';
COMMENT ON COLUMN events.title IS '预测事件标题';
COMMENT ON COLUMN events.type IS '事件类型：sports=体育，politics=政治，economy=经济，other=其他';
COMMENT ON COLUMN events.platform_id IS '关联第三方平台ID';
COMMENT ON COLUMN events.platform_event_id IS '第三方平台原生事件ID';
COMMENT ON COLUMN events.canonical_key IS '聚合键，用于同场多平台归并';
COMMENT ON COLUMN events.start_time IS '事件开始时间';
COMMENT ON COLUMN events.end_time IS '事件结束时间';
COMMENT ON COLUMN events.resolve_time IS '事件结果公布时间';
COMMENT ON COLUMN events.options IS '事件下注选项（JSON：如 {"yes":"发生","no":"不发生"}）';
COMMENT ON COLUMN events.result IS '事件最终结果（对应 options 中的 key）';
COMMENT ON COLUMN events.result_source IS '结果来源：oracle/platform/manual';
COMMENT ON COLUMN events.result_verified IS '结果是否多源核验';
COMMENT ON COLUMN events.status IS '事件状态：active/resolved/canceled';
COMMENT ON COLUMN events.is_hot IS '是否为热门事件（优先缓存）';
COMMENT ON COLUMN events.created_at IS '事件录入时间';
COMMENT ON COLUMN events.updated_at IS '事件信息更新时间';
CREATE INDEX IF NOT EXISTS idx_events_platform_id ON events(platform_id);
CREATE INDEX IF NOT EXISTS idx_events_platform_event_id ON events(platform_event_id);
CREATE INDEX IF NOT EXISTS idx_events_canonical_key ON events(canonical_key);
CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
CREATE INDEX IF NOT EXISTS idx_events_end_time ON events(end_time);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
CREATE INDEX IF NOT EXISTS idx_events_is_hot ON events(is_hot);
CREATE INDEX IF NOT EXISTS idx_events_updated_at ON events(updated_at);
CREATE INDEX IF NOT EXISTS idx_events_title_gin ON events USING GIN(to_tsvector('english', title));

-- ------------------------------
-- 4. 事件赔率表（event_odds）
-- ------------------------------
CREATE TABLE IF NOT EXISTS event_odds (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
    unique_event_platform VARCHAR(128) NOT NULL UNIQUE,
    platform_id BIGINT NOT NULL,
    option_name VARCHAR(64) NOT NULL,
    option_type VARCHAR(16),
    price DECIMAL(10,2) NOT NULL,
    liquidity DECIMAL(10,2) DEFAULT 0,
    volume DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
COMMENT ON TABLE event_odds IS '事件赔率表，存储各平台各事件的实时赔率数据';
COMMENT ON COLUMN event_odds.id IS '自增主键ID';
COMMENT ON COLUMN event_odds.event_id IS '关联预测事件ID';
COMMENT ON COLUMN event_odds.unique_event_platform IS '事件+平台唯一标识';
COMMENT ON COLUMN event_odds.platform_id IS '关联第三方平台ID';
COMMENT ON COLUMN event_odds.option_name IS '赔率选项名称（如 yes/no）';
COMMENT ON COLUMN event_odds.option_type IS '归一化选项：win/draw/lose';
COMMENT ON COLUMN event_odds.price IS '赔率价格';
COMMENT ON COLUMN event_odds.liquidity IS '流动性';
COMMENT ON COLUMN event_odds.volume IS '交易量';
COMMENT ON COLUMN event_odds.updated_at IS '赔率更新时间（校验时效性）';
COMMENT ON COLUMN event_odds.deleted_at IS '软删除时间';
CREATE INDEX IF NOT EXISTS idx_event_odds_event_id ON event_odds(event_id);
CREATE INDEX IF NOT EXISTS idx_event_odds_platform_id ON event_odds(platform_id);
CREATE INDEX IF NOT EXISTS idx_event_odds_updated_at ON event_odds(updated_at);
CREATE INDEX IF NOT EXISTS idx_event_odds_deleted_at ON event_odds(deleted_at);

-- ------------------------------
-- 5. 用户订单表（orders）
-- ------------------------------
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    order_uuid VARCHAR(64) NOT NULL UNIQUE,
    user_wallet VARCHAR(64) NOT NULL REFERENCES users(wallet_address),
    event_id BIGINT NOT NULL REFERENCES events(id),
    platform_id BIGINT NOT NULL REFERENCES platforms(id),
    platform_order_id VARCHAR(64),
    bet_option VARCHAR(32) NOT NULL,
    bet_amount NUMERIC(18,6) NOT NULL,
    locked_odds NUMERIC(10,2) NOT NULL,
    expected_profit NUMERIC(18,6) DEFAULT 0,
    actual_profit NUMERIC(18,6) DEFAULT 0,
    platform_fee NUMERIC(18,6) DEFAULT 0,
    manage_fee NUMERIC(18,6) DEFAULT 0,
    gas_fee NUMERIC(18,6) DEFAULT 0,
    fund_lock_tx_hash VARCHAR(66),
    settlement_tx_hash VARCHAR(66),
    status VARCHAR(16) DEFAULT 'pending_lock',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
COMMENT ON TABLE orders IS '用户订单表，存储用户所有下注订单信息';
COMMENT ON COLUMN orders.order_uuid IS '全局唯一订单ID';
COMMENT ON COLUMN orders.user_wallet IS '关联用户钱包地址';
COMMENT ON COLUMN orders.event_id IS '关联预测事件ID';
COMMENT ON COLUMN orders.platform_id IS '下注的第三方平台ID';
COMMENT ON COLUMN orders.platform_order_id IS '第三方平台原生订单号';
COMMENT ON COLUMN orders.bet_option IS '用户下注选项（对应 events.options 的 key）';
COMMENT ON COLUMN orders.bet_amount IS '用户下注金额（USDC）';
COMMENT ON COLUMN orders.locked_odds IS '下单时锁定的赔率';
COMMENT ON COLUMN orders.expected_profit IS '预期收益（USDC）';
COMMENT ON COLUMN orders.actual_profit IS '实际收益（USDC，亏损为负）';
COMMENT ON COLUMN orders.platform_fee IS '第三方平台手续费（USDC）';
COMMENT ON COLUMN orders.manage_fee IS '平台1%管理费（USDC）';
COMMENT ON COLUMN orders.gas_fee IS '链上Gas费（换算为USDC）';
COMMENT ON COLUMN orders.fund_lock_tx_hash IS '资金锁定交易哈希（0x开头）';
COMMENT ON COLUMN orders.settlement_tx_hash IS '结算交易哈希（0x开头）';
COMMENT ON COLUMN orders.status IS '订单状态：pending_lock/locked/placed/settlable/settled/abnormal/refunded';
COMMENT ON COLUMN orders.created_at IS '订单创建时间';
COMMENT ON COLUMN orders.updated_at IS '订单状态更新时间';
CREATE INDEX IF NOT EXISTS idx_orders_user_wallet ON orders(user_wallet);
CREATE INDEX IF NOT EXISTS idx_orders_event_id ON orders(event_id);
CREATE INDEX IF NOT EXISTS idx_orders_platform_id ON orders(platform_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

-- ------------------------------
-- 6. 链上事件记录表（contract_events）
-- ------------------------------
CREATE TABLE IF NOT EXISTS contract_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(32) NOT NULL,
    order_uuid VARCHAR(64) REFERENCES orders(order_uuid),
    user_wallet VARCHAR(64) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL UNIQUE,
    block_number BIGINT,
    event_data JSONB NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
COMMENT ON TABLE contract_events IS '链上事件记录表，用于后端监听和追溯；order_uuid 可空，BetPlaced 先入库再回写';
COMMENT ON COLUMN contract_events.event_type IS '链上事件类型：FundLocked/SettlementCompleted/FundUnlocked 等';
COMMENT ON COLUMN contract_events.order_uuid IS '关联订单UUID（可空，先记链上事件再创建订单后回写）';
COMMENT ON COLUMN contract_events.user_wallet IS '用户钱包地址';
COMMENT ON COLUMN contract_events.tx_hash IS '链上交易哈希（0x开头，唯一）';
COMMENT ON COLUMN contract_events.block_number IS '区块高度';
COMMENT ON COLUMN contract_events.event_data IS '事件原始数据（JSON）';
COMMENT ON COLUMN contract_events.processed IS '是否已处理';
COMMENT ON COLUMN contract_events.processed_at IS '处理时间';
COMMENT ON COLUMN contract_events.created_at IS '链上事件发生时间';
CREATE INDEX IF NOT EXISTS idx_contract_events_order_uuid ON contract_events(order_uuid);
CREATE INDEX IF NOT EXISTS idx_contract_events_user_wallet ON contract_events(user_wallet);
CREATE INDEX IF NOT EXISTS idx_contract_events_event_type ON contract_events(event_type);
CREATE INDEX IF NOT EXISTS idx_contract_events_processed ON contract_events(processed);
CREATE INDEX IF NOT EXISTS idx_contract_events_created_at ON contract_events(created_at);
CREATE INDEX IF NOT EXISTS idx_contract_events_event_data_gin ON contract_events USING GIN(event_data);

-- ------------------------------
-- 7. 结算记录表（settlement_records）
-- ------------------------------
CREATE TABLE IF NOT EXISTS settlement_records (
    id BIGSERIAL PRIMARY KEY,
    order_uuid VARCHAR(64) NOT NULL REFERENCES orders(order_uuid),
    user_wallet VARCHAR(64) NOT NULL,
    settlement_amount NUMERIC(18,6) NOT NULL,
    manage_fee NUMERIC(18,6) DEFAULT 0,
    gas_fee NUMERIC(18,6) DEFAULT 0,
    tx_hash VARCHAR(66) NOT NULL UNIQUE,
    settlement_time TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);
COMMENT ON TABLE settlement_records IS '用户结算记录表，用于审计';
COMMENT ON COLUMN settlement_records.order_uuid IS '关联订单UUID';
COMMENT ON COLUMN settlement_records.user_wallet IS '用户钱包地址';
COMMENT ON COLUMN settlement_records.settlement_amount IS '用户实际到账金额（USDC）';
COMMENT ON COLUMN settlement_records.manage_fee IS '结算时扣除的1%管理费（USDC）';
COMMENT ON COLUMN settlement_records.gas_fee IS '结算时用户支付的Gas费（USDC）';
COMMENT ON COLUMN settlement_records.tx_hash IS '结算交易哈希（0x开头）';
COMMENT ON COLUMN settlement_records.settlement_time IS '实际结算时间';
COMMENT ON COLUMN settlement_records.created_at IS '记录创建时间';
CREATE INDEX IF NOT EXISTS idx_settlement_records_order_uuid ON settlement_records(order_uuid);
CREATE INDEX IF NOT EXISTS idx_settlement_records_user_wallet ON settlement_records(user_wallet);
CREATE INDEX IF NOT EXISTS idx_settlement_records_settlement_time ON settlement_records(settlement_time);

-- ------------------------------
-- 8. 聚合赛事主表（canonical_events）
-- ------------------------------
CREATE TABLE IF NOT EXISTS canonical_events (
    id BIGSERIAL PRIMARY KEY,
    sport_type VARCHAR(64) NOT NULL,
    title VARCHAR(256) NOT NULL,
    home_team VARCHAR(128),
    away_team VARCHAR(128),
    match_time TIMESTAMP NOT NULL,
    canonical_key VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(16) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
COMMENT ON TABLE canonical_events IS '聚合赛事主表，同一场比赛多平台去重后一条；id 即 canonical_id';
COMMENT ON COLUMN canonical_events.sport_type IS '运动/赛事类型';
COMMENT ON COLUMN canonical_events.title IS '赛事标题';
COMMENT ON COLUMN canonical_events.home_team IS '主队';
COMMENT ON COLUMN canonical_events.away_team IS '客队';
COMMENT ON COLUMN canonical_events.match_time IS '比赛时间';
COMMENT ON COLUMN canonical_events.canonical_key IS '规范化键，用于同场判定';
COMMENT ON COLUMN canonical_events.status IS '状态：active 等';

-- ------------------------------
-- 9. 聚合赛事-平台事件映射（event_platform_links）
-- ------------------------------
CREATE TABLE IF NOT EXISTS event_platform_links (
    id BIGSERIAL PRIMARY KEY,
    canonical_event_id BIGINT NOT NULL REFERENCES canonical_events(id),
    event_id BIGINT NOT NULL REFERENCES events(id),
    platform_id BIGINT NOT NULL REFERENCES platforms(id),
    CONSTRAINT uq_canonical_platform UNIQUE (canonical_event_id, platform_id)
);
COMMENT ON TABLE event_platform_links IS '聚合赛事与平台事件映射';

-- ------------------------------
-- 触发器：自动更新 updated_at
-- ------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
COMMENT ON FUNCTION update_updated_at_column IS '自动更新表的 updated_at 字段为当前时间';

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_platforms_updated_at ON platforms;
CREATE TRIGGER update_platforms_updated_at BEFORE UPDATE ON platforms FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_events_updated_at ON events;
CREATE TRIGGER update_events_updated_at BEFORE UPDATE ON events FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_event_odds_updated_at ON event_odds;
CREATE TRIGGER update_event_odds_updated_at BEFORE UPDATE ON event_odds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_canonical_events_updated_at ON canonical_events;
CREATE TRIGGER update_canonical_events_updated_at BEFORE UPDATE ON canonical_events FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

## 前置准备
- 1. Postgres 服务（版本建议 11+）
- 2. **应用启动时会自动创建不存在的数据库 `forecast_aggregation`**（需能连上默认库 `postgres`），并自动检查、创建不存在的表；无需预先建库建表
- 3. 若需手动初始化或与 Go 模型完全一致（含注释、索引、触发器），可先建库再在该库中执行上文「库表结构」中的完整 SQL

## 快速启动

- 1. 配置敏感信息（不提交 git）：复制 `.env.example` 为 `.env`，填入真实值
```bash
cp .env.example .env
# 编辑 .env，填写 KALSHI_AUTH_KEY、KALSHI_AUTH_SECRET 等
```

- 2. 修改配置文件 config/config.yaml 以下配置（非敏感部分）
```yaml
# 数据库配置
mysql:
  #使用的是postgres,请修改成自己环境下postgres的配置
  dsn: "postgres://postgres:postgres@127.0.0.1:5433/forecast_aggregation?sslmode=disable&TimeZone=Asia/Shanghai"

# 各平台独立配置
platforms:
  # Polymarket配置
  polymarket:
    base_url: "https://gamma-api.polymarket.com"
    protocol: "rest"
    timeout: 10
    retry_count: 2
    auth_token: ""
    #代理地址 注意：因为polymarket是国外的，需要开代理才能访问，工程请求默认不会被服务器开启的梯子代理，因此需要指定代理地址
    #如果本地可以直接访问外网，这这个proxy:直接置空即可
    proxy: "http://127.0.0.1:7890"
    # 最小下注金额
    min_bet: 1
    # 最大下注金额
    max_bet: 1

  kalshi:
    base_url: "https://trading-api.kalshi.com/trade-api/v2" # Kalshi官方基础URL
    sport_path: "/markets" # 市场数据接口路径
    protocol: "rest"
    timeout: 10 # 超时时间（匹配Kalshi建议）
    retry_count: 3 # 重试次数
    # auth_key、auth_secret 从 .env 读取（KALSHI_AUTH_KEY、KALSHI_AUTH_SECRET），此处留空即可
    #代理地址 根据实际情况配置
    proxy: "127.0.0.1:7890"
    # 最小下注金额
    min_bet: 1
    # 最大下注金额
    max_bet: 1
```
- 3. 执行启动命令
```shell
go run cmd/main.go
```
出现以下日志说明启动成功
```text
time="2026-02-08T18:19:29+08:00" level=info msg="配置文件加载成功"
time="2026-02-08T18:19:29+08:00" level=info msg="PostgreSQL连接成功"
time="2026-02-08T18:19:29+08:00" level=info msg="Gin运行模式: debug"
time="2026-02-08T18:19:29+08:00" level=info msg="服务启动成功，端口：8081"
```

- 4. 执行以下命令触发同步指定预测平台的数据
```shell
curl --location --request POST 'localhost:8081/sync/platform/polymarket' \
--data ''
```