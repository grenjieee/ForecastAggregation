## 代码结构 
```text
ForecastSync/
├── cmd/
│   └── server/
│       └── main.go          # 入口文件（启动API服务）
├── internal/
│   ├── adapter/             # 平台适配器层（每个平台一个目录）
│   │   ├── polymarket/      # Polymarket适配器
│   │   │   ├── adapter.go   # 实现通用接口
│   │   │   ├── client.go    # HTTP客户端
│   │   │   └── model.go     # Polymarket原生模型
│   │   └── kalshi/          # Kalshi适配器
│   │       ├── adapter.go   # 实现通用接口
│   │       ├── client.go    # HTTP客户端
│   │       └── model.go     # Kalshi原生模型
│   ├── config/              # 配置管理
│   │   └── config.go        # 全局配置
│   ├── interfaces/          # 通用接口定义
│   │   └── platform.go      # 平台适配器通用接口
│   ├── model/               # 数据库模型 + 通用数据结构
│   │   ├── db.go            # 数据库表模型（Event/EventOdds等）
│   │   └── platform.go      # 通用平台数据结构
│   ├── repository/          # 通用数据库操作
│   │   └── event.go         # Event/EventOdds入库逻辑
│   ├── service/             # 通用同步服务
│   │   └── sync.go          # 多平台同步核心逻辑
│   └── api/                 # API接口层
│       └── sync_handler.go  # 同步接口
├── go.mod
└── go.sum
```

## 库表结构
```sql
-- 创建数据库
CREATE DATABASE forecast_aggregation
  WITH 
  OWNER = postgres
  ENCODING = 'UTF8'
  LC_COLLATE = 'en_US.UTF-8'
  LC_CTYPE = 'en_US.UTF-8'
  TABLESPACE = pg_default
  CONNECTION LIMIT = -1
  COMMENT = '预测市场聚合套利平台核心数据库';

SET TIME ZONE 'UTC';

-- 前提：已创建forecast_aggregation数据库并切换到该库，且设置了UTC时区
-- 以下SQL需在forecast_aggregation数据库的查询工具中执行

-- ------------------------------
-- 1. 用户表（users）
-- ------------------------------
CREATE TABLE users (
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
-- 表备注
COMMENT ON TABLE users IS '用户基础信息表，关联钱包地址作为唯一标识';
-- 字段备注
COMMENT ON COLUMN users.id IS '自增主键ID';
COMMENT ON COLUMN users.wallet_address IS '用户钱包地址（0x开头，小写存储）';
COMMENT ON COLUMN users.total_profit IS '用户累计盈利（USDC，保留6位小数）';
COMMENT ON COLUMN users.total_loss IS '用户累计亏损（USDC，保留6位小数）';
COMMENT ON COLUMN users.total_fee IS '用户累计支付的1%平台管理费（USDC）';
COMMENT ON COLUMN users.gas_fee_total IS '用户累计支付的链上Gas费（换算为USDC）';
COMMENT ON COLUMN users.is_active IS '用户是否活跃：true=活跃，false=禁用';
COMMENT ON COLUMN users.created_at IS '用户创建时间（首次登录时间）';
COMMENT ON COLUMN users.updated_at IS '用户信息更新时间';
-- 索引+索引备注
CREATE INDEX idx_users_created_at ON users(created_at);
COMMENT ON INDEX idx_users_created_at IS '用户创建时间索引，用于按时间筛选用户';

-- ------------------------------
-- 2. 第三方平台配置表（platforms）
-- ------------------------------
CREATE TABLE platforms (
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
-- 表备注
COMMENT ON TABLE platforms IS '第三方预测平台配置表，管理多平台对接参数';
-- 字段备注
COMMENT ON COLUMN platforms.id IS '自增主键ID';
COMMENT ON COLUMN platforms.name IS '第三方预测平台名称（如Polymarket、Manifold）';
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
-- 索引+索引备注
CREATE INDEX idx_platforms_name ON platforms(name);
COMMENT ON INDEX idx_platforms_name IS '平台名称索引，快速查询指定平台配置';
CREATE INDEX idx_platforms_type ON platforms(type);
COMMENT ON INDEX idx_platforms_type IS '平台类型索引，区分链上/中心化平台';
CREATE INDEX idx_platforms_is_hot ON platforms(is_hot);
COMMENT ON INDEX idx_platforms_is_hot IS '热门平台索引，优先缓存热门平台数据';
CREATE INDEX idx_platforms_is_enabled ON platforms(is_enabled);
COMMENT ON INDEX idx_platforms_is_enabled IS '启用状态索引，过滤禁用平台';
-- 初始化平台数据 ---
INSERT INTO `platforms` (`id`, `name`, `type`, `api_url`, `contract_address`, `rpc_url`, `api_key`, `api_limit`, `current_api_usage`, `is_hot`, `is_enabled`, `created_at`, `updated_at`) VALUES('1','polymarket','centralized','https://gamma-api.polymarket.com',NULL,NULL,NULL,'600','0','0','1','2026-02-08 18:05:07','2026-02-08 18:05:10');
INSERT INTO `platforms` (`id`, `name`, `type`, `api_url`, `contract_address`, `rpc_url`, `api_key`, `api_limit`, `current_api_usage`, `is_hot`, `is_enabled`, `created_at`, `updated_at`) VALUES('2','kalshi','centralized','https://api.kalshi.com/v1',NULL,NULL,NULL,'600','0','0','1','2026-02-08 18:06:34','2026-02-08 18:06:39');

-- ------------------------------
-- 3. 预测事件主表（events）
-- ------------------------------
CREATE TABLE events (
                        id BIGSERIAL PRIMARY KEY,
                        event_uuid VARCHAR(64) NOT NULL UNIQUE,
                        title VARCHAR(256) NOT NULL,
                        type VARCHAR(16) NOT NULL,
                        platform_id BIGINT NOT NULL REFERENCES platforms(id),
                        platform_event_id VARCHAR(64) NOT NULL,
                        start_time TIMESTAMP NOT NULL,
                        end_time TIMESTAMP NOT NULL,
                        resolve_time TIMESTAMP,
                        options JSONB NOT NULL,
                        result VARCHAR(32),
                        result_source VARCHAR(64),
                        result_verified BOOLEAN DEFAULT FALSE,
                        status VARCHAR(16) DEFAULT 'active',
                        is_hot BOOLEAN DEFAULT FALSE,
                        created_at TIMESTAMP DEFAULT NOW(),
                        updated_at TIMESTAMP DEFAULT NOW()
);
-- 表备注
COMMENT ON TABLE events IS '预测事件主表，存储所有对接平台的预测事件信息';
-- 字段备注
COMMENT ON COLUMN events.id IS '自增主键ID';
COMMENT ON COLUMN events.event_uuid IS '全局唯一事件ID（规则：平台ID+平台原生事件ID）';
COMMENT ON COLUMN events.title IS '预测事件标题（如「2024世界杯冠军：阿根廷」）';
COMMENT ON COLUMN events.type IS '事件类型：sports=体育，politics=政治，economy=经济，other=其他';
COMMENT ON COLUMN events.platform_id IS '关联第三方平台ID';
COMMENT ON COLUMN events.platform_event_id IS '第三方平台原生事件ID';
COMMENT ON COLUMN events.start_time IS '事件开始时间';
COMMENT ON COLUMN events.end_time IS '事件结束时间';
COMMENT ON COLUMN events.resolve_time IS '事件结果公布时间';
COMMENT ON COLUMN events.options IS '事件下注选项（JSON格式：{"yes":"发生","no":"不发生"}）';
COMMENT ON COLUMN events.result IS '事件最终结果（对应options中的key，如yes/no）';
COMMENT ON COLUMN events.result_source IS '结果来源：oracle=预言机，platform=平台官方，manual=人工核验';
COMMENT ON COLUMN events.result_verified IS '结果是否多源核验：true=是，false=否';
COMMENT ON COLUMN events.status IS '事件状态：active=进行中，resolved=已出结果，canceled=已取消';
COMMENT ON COLUMN events.is_hot IS '是否为热门事件：true=是，false=否（优先内存缓存）';
COMMENT ON COLUMN events.created_at IS '事件录入时间';
COMMENT ON COLUMN events.updated_at IS '事件信息更新时间';
-- 索引+索引备注
CREATE INDEX idx_events_platform_id ON events(platform_id);
COMMENT ON INDEX idx_events_platform_id IS '平台ID索引，查询指定平台的事件';
CREATE INDEX idx_events_platform_event_id ON events(platform_event_id);
COMMENT ON INDEX idx_events_platform_event_id IS '平台原生事件ID索引，关联第三方平台事件';
CREATE INDEX idx_events_start_time ON events(start_time);
COMMENT ON INDEX idx_events_start_time IS '事件开始时间索引，筛选未开始事件';
CREATE INDEX idx_events_end_time ON events(end_time);
COMMENT ON INDEX idx_events_end_time IS '事件结束时间索引，筛选进行中/已结束事件';
CREATE INDEX idx_events_status ON events(status);
COMMENT ON INDEX idx_events_status IS '事件状态索引，筛选进行中/已出结果事件';
CREATE INDEX idx_events_is_hot ON events(is_hot);
COMMENT ON INDEX idx_events_is_hot IS '热门事件索引，优先缓存热门事件';
CREATE INDEX idx_events_updated_at ON events(updated_at);
COMMENT ON INDEX idx_events_updated_at IS '更新时间索引，校验赔率数据时效性';
CREATE INDEX idx_events_title_gin ON events USING GIN(to_tsvector('english', title));
COMMENT ON INDEX idx_events_title_gin IS '事件标题全文索引，支持模糊搜索';

-- ------------------------------
-- 4. 事件赔率表（event_odds）- 修正唯一约束命名
-- ------------------------------
CREATE TABLE event_odds (
                            id BIGSERIAL PRIMARY KEY,
                            event_id BIGINT NOT NULL REFERENCES events(id),
                            platform_id BIGINT NOT NULL REFERENCES platforms(id),
                            odds JSONB NOT NULL,
                            fee NUMERIC(8,4) DEFAULT 0,
                            max_bet NUMERIC(18,6),
                            min_bet NUMERIC(18,6) DEFAULT 0.1,
                            locked_odds JSONB,
                            cache_level VARCHAR(8) DEFAULT 'db',
                            updated_at TIMESTAMP DEFAULT NOW(),
    -- 显式命名唯一约束，解决备注报错问题
                            CONSTRAINT uk_event_platform UNIQUE (event_id, platform_id)
);
-- 表备注
COMMENT ON TABLE event_odds IS '事件赔率表，存储各平台各事件的实时赔率数据';
-- 字段备注
COMMENT ON COLUMN event_odds.id IS '自增主键ID';
COMMENT ON COLUMN event_odds.event_id IS '关联预测事件ID';
COMMENT ON COLUMN event_odds.platform_id IS '关联第三方平台ID';
COMMENT ON COLUMN event_odds.odds IS '事件各选项赔率（JSON格式：{"yes":9.0,"no":1.2}）';
COMMENT ON COLUMN event_odds.fee IS '平台手续费比例（如0.02=2%）';
COMMENT ON COLUMN event_odds.max_bet IS '平台允许的最大下注金额（USDC）';
COMMENT ON COLUMN event_odds.min_bet IS '平台允许的最小下注金额（USDC）';
COMMENT ON COLUMN event_odds.locked_odds IS '历史锁定赔率快照（下单时的赔率）';
COMMENT ON COLUMN event_odds.cache_level IS '缓存层级：mem=内存缓存，db=数据库缓存';
COMMENT ON COLUMN event_odds.updated_at IS '赔率更新时间（校验时效性依据）';
-- 唯一约束备注（现在能正常生效）
COMMENT ON CONSTRAINT uk_event_platform ON event_odds IS '唯一约束：同一事件+同一平台仅一条赔率记录';
-- 索引+索引备注
CREATE INDEX idx_event_odds_event_id ON event_odds(event_id);
COMMENT ON INDEX idx_event_odds_event_id IS '事件ID索引，查询指定事件的所有平台赔率';
CREATE INDEX idx_event_odds_platform_id ON event_odds(platform_id);
COMMENT ON INDEX idx_event_odds_platform_id IS '平台ID索引，查询指定平台的所有赔率';
CREATE INDEX idx_event_odds_updated_at ON event_odds(updated_at);
COMMENT ON INDEX idx_event_odds_updated_at IS '更新时间索引，校验赔率数据是否过期';
CREATE INDEX idx_event_odds_odds_gin ON event_odds USING GIN(odds);
COMMENT ON INDEX idx_event_odds_odds_gin IS '赔率JSONB索引，支持按赔率值筛选';

-- ------------------------------
-- 5. 用户订单表（orders）
-- ------------------------------
CREATE TABLE orders (
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
-- 表备注
COMMENT ON TABLE orders IS '用户订单表，存储用户所有下注订单信息';
-- 字段备注
COMMENT ON COLUMN orders.id IS '自增主键ID';
COMMENT ON COLUMN orders.order_uuid IS '全局唯一订单ID（规则：钱包地址+时间戳+随机数）';
COMMENT ON COLUMN orders.user_wallet IS '关联用户钱包地址';
COMMENT ON COLUMN orders.event_id IS '关联预测事件ID';
COMMENT ON COLUMN orders.platform_id IS '下注的第三方平台ID';
COMMENT ON COLUMN orders.platform_order_id IS '第三方平台原生订单号';
COMMENT ON COLUMN orders.bet_option IS '用户下注选项（对应events.options中的key）';
COMMENT ON COLUMN orders.bet_amount IS '用户下注金额（USDC）';
COMMENT ON COLUMN orders.locked_odds IS '下单时锁定的赔率（固定不变）';
COMMENT ON COLUMN orders.expected_profit IS '预期收益（USDC）';
COMMENT ON COLUMN orders.actual_profit IS '实际收益（USDC，亏损为负数）';
COMMENT ON COLUMN orders.platform_fee IS '第三方平台收取的手续费（USDC）';
COMMENT ON COLUMN orders.manage_fee IS '平台收取的1%管理费（仅盈利时扣除，USDC）';
COMMENT ON COLUMN orders.gas_fee IS '用户支付的链上Gas费（换算为USDC）';
COMMENT ON COLUMN orders.fund_lock_tx_hash IS '资金锁定的链上交易哈希（0x开头）';
COMMENT ON COLUMN orders.settlement_tx_hash IS '结算的链上交易哈希（0x开头）';
COMMENT ON COLUMN orders.status IS '订单状态：pending_lock=待锁定资金,locked=已锁定资金,placed=已下注成功,settlable=可结算,settled=已结算,abnormal=异常,refunded=已退款';
COMMENT ON COLUMN orders.created_at IS '订单创建时间';
COMMENT ON COLUMN orders.updated_at IS '订单状态更新时间';
-- 索引+索引备注
CREATE INDEX idx_orders_user_wallet ON orders(user_wallet);
COMMENT ON INDEX idx_orders_user_wallet IS '用户钱包索引，查询指定用户的所有订单';
CREATE INDEX idx_orders_event_id ON orders(event_id);
COMMENT ON INDEX idx_orders_event_id IS '事件ID索引，查询指定事件的所有订单';
CREATE INDEX idx_orders_platform_id ON orders(platform_id);
COMMENT ON INDEX idx_orders_platform_id IS '平台ID索引，查询指定平台的订单';
CREATE INDEX idx_orders_status ON orders(status);
COMMENT ON INDEX idx_orders_status IS '订单状态索引，筛选待结算/已结算订单';
CREATE INDEX idx_orders_created_at ON orders(created_at);
COMMENT ON INDEX idx_orders_created_at IS '创建时间索引，按时间筛选订单';

-- ------------------------------
-- 6. 链上事件记录表（contract_events）
-- ------------------------------
CREATE TABLE contract_events (
                                 id BIGSERIAL PRIMARY KEY,
                                 event_type VARCHAR(32) NOT NULL,
                                 order_uuid VARCHAR(64) NOT NULL REFERENCES orders(order_uuid),
                                 user_wallet VARCHAR(64) NOT NULL,
                                 tx_hash VARCHAR(66) NOT NULL UNIQUE,
                                 block_number BIGINT,
                                 event_data JSONB NOT NULL,
                                 processed BOOLEAN DEFAULT FALSE,
                                 processed_at TIMESTAMP,
                                 created_at TIMESTAMP DEFAULT NOW()
);
-- 表备注
COMMENT ON TABLE contract_events IS '链上事件记录表，留存智能合约关键操作痕迹，用于后端监听和追溯';
-- 字段备注
COMMENT ON COLUMN contract_events.id IS '自增主键ID';
COMMENT ON COLUMN contract_events.event_type IS '链上事件类型：FundLocked=资金锁定成功,SettlementCompleted=结算完成,FundUnlocked=资金解锁（退款）';
COMMENT ON COLUMN contract_events.order_uuid IS '关联订单UUID';
COMMENT ON COLUMN contract_events.user_wallet IS '关联用户钱包地址';
COMMENT ON COLUMN contract_events.tx_hash IS '链上交易哈希（0x开头，唯一标识）';
COMMENT ON COLUMN contract_events.block_number IS '交易所在区块高度';
COMMENT ON COLUMN contract_events.event_data IS '事件原始数据（JSON格式，解析后的合约参数）';
COMMENT ON COLUMN contract_events.processed IS '后端是否已处理该事件：true=已处理，false=未处理';
COMMENT ON COLUMN contract_events.processed_at IS '后端处理事件的时间';
COMMENT ON COLUMN contract_events.created_at IS '链上事件发生时间';
-- 索引+索引备注
CREATE INDEX idx_contract_events_order_uuid ON contract_events(order_uuid);
COMMENT ON INDEX idx_contract_events_order_uuid IS '订单UUID索引，关联订单事件';
CREATE INDEX idx_contract_events_user_wallet ON contract_events(user_wallet);
COMMENT ON INDEX idx_contract_events_user_wallet IS '用户钱包索引，查询指定用户的链上事件';
CREATE INDEX idx_contract_events_event_type ON contract_events(event_type);
COMMENT ON INDEX idx_contract_events_event_type IS '事件类型索引，筛选资金锁定/结算事件';
CREATE INDEX idx_contract_events_processed ON contract_events(processed);
COMMENT ON INDEX idx_contract_events_processed IS '处理状态索引，筛选未处理事件';
CREATE INDEX idx_contract_events_created_at ON contract_events(created_at);
COMMENT ON INDEX idx_contract_events_created_at IS '事件发生时间索引，按时间筛选事件';
CREATE INDEX idx_contract_events_event_data_gin ON contract_events USING GIN(event_data);
COMMENT ON INDEX idx_contract_events_event_data_gin IS '事件数据JSONB索引，支持复杂查询';

-- ------------------------------
-- 7. 结算记录表（settlement_records）
-- ------------------------------
CREATE TABLE settlement_records (
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
-- 表备注
COMMENT ON TABLE settlement_records IS '用户结算记录表，留存详细的结算金额和手续费信息，用于审计';
-- 字段备注
COMMENT ON COLUMN settlement_records.id IS '自增主键ID';
COMMENT ON COLUMN settlement_records.order_uuid IS '关联订单UUID';
COMMENT ON COLUMN settlement_records.user_wallet IS '关联用户钱包地址';
COMMENT ON COLUMN settlement_records.settlement_amount IS '用户实际到账金额（USDC）';
COMMENT ON COLUMN settlement_records.manage_fee IS '结算时扣除的1%管理费（USDC）';
COMMENT ON COLUMN settlement_records.gas_fee IS '结算时用户支付的Gas费（换算为USDC）';
COMMENT ON COLUMN settlement_records.tx_hash IS '结算交易哈希（0x开头）';
COMMENT ON COLUMN settlement_records.settlement_time IS '实际结算时间';
COMMENT ON COLUMN settlement_records.created_at IS '记录创建时间';
-- 索引+索引备注
CREATE INDEX idx_settlement_records_order_uuid ON settlement_records(order_uuid);
COMMENT ON INDEX idx_settlement_records_order_uuid IS '订单UUID索引，关联订单结算记录';
CREATE INDEX idx_settlement_records_user_wallet ON settlement_records(user_wallet);
COMMENT ON INDEX idx_settlement_records_user_wallet IS '用户钱包索引，查询指定用户的结算记录';
CREATE INDEX idx_settlement_records_settlement_time ON settlement_records(settlement_time);
COMMENT ON INDEX idx_settlement_records_settlement_time IS '结算时间索引，按时间筛选结算记录';

-- ------------------------------
-- 触发器：自动更新updated_at字段
-- ------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- 触发器函数备注
COMMENT ON FUNCTION update_updated_at_column IS '自动更新表的updated_at字段为当前时间';

-- 为各表添加更新触发器
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
COMMENT ON TRIGGER update_users_updated_at ON users IS '用户表更新时自动刷新updated_at';

CREATE TRIGGER update_platforms_updated_at
    BEFORE UPDATE ON platforms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
COMMENT ON TRIGGER update_platforms_updated_at ON platforms IS '平台配置表更新时自动刷新updated_at';

CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
COMMENT ON TRIGGER update_events_updated_at ON events IS '事件表更新时自动刷新updated_at';

CREATE TRIGGER update_event_odds_updated_at
    BEFORE UPDATE ON event_odds
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
COMMENT ON TRIGGER update_event_odds_updated_at ON event_odds IS '赔率表更新时自动刷新updated_at';

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
COMMENT ON TRIGGER update_orders_updated_at ON orders IS '订单表更新时自动刷新updated_at';
```

## 前置准备
- 1.Postgres数据库
- 2.在Postgres中建立数据库forecast_aggregation 并执行库表结构中的SQL

## 快速启动

- 1.修改配置文件 config/config.yaml 以下配置
```toml
# 数据库配置
mysql:
  #使用的是postgres,请修改成自己环境下postgres的配置
  dsn: "postgres://postgres:postgres@192.168.1.37:5432/forecast_aggregation?sslmode=disable&TimeZone=Asia/Shanghai"

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
    auth_key: "YOUR_KALSHI_API_KEY" # 替换为你的Kalshi API Key
    auth_secret: "YOUR_KALSHI_API_SECRET" # 替换为你的Kalshi API Secret
    #代理地址 根据实际情况配置
    proxy: "127.0.0.1:7890"
    # 最小下注金额
    min_bet: 1
    # 最大下注金额
    max_bet: 1
```
- 2.执行启动命令
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

- 3.执行以下命令触发同步指定预测平台的数据
```shell
curl --location --request POST 'localhost:8081/sync/platform/polymarket' \
--data ''
```