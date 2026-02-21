# ForecastSync API 说明

## 市场

### 1. 查询市场列表

市场列表，分页。

- **接口 path:** `GET /api/markets`
- **接口协议:** HTTP GET

#### 接口请求参数

| 请求参数  | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------- | -------- | -------- | ------ | ---- |
| status    | string   | 否       | active | active: 当前可下注; resolved: 已结束 |
| type      | string   | 否       | sports | 市场类型，一期固定 sports |
| page      | int      | 否       | 1      | 当前查询页数 |
| page_size | int      | 否       | 20     | 每页返回的记录数 |

#### 接口响应参数

| 参数名    | 字段类型       | 是否可空 | 默认值 | 备注 |
| --------- | -------------- | -------- | ------ | ---- |
| page      | int            | 否       | 0      | 当前页数 |
| page_size | int            | 否       | 0      | 当前页返回的记录数 |
| total     | int64          | 否       | 0      | 总符合条件数 |
| items     | []MarketSummary| 是       | 空     | 符合条件的市场列表 |

#### MarketSummary 子结构

| 参数名              | 字段类型     | 是否可空 | 备注 |
| ------------------- | ------------ | -------- | ---- |
| canonical_id        | int64        | 否       | 聚合赛事 ID，Compare 链接用 |
| title               | string       | 否       | 市场标题 |
| description         | string       | 否       | 详细描述 |
| type                | string       | 否       | 一期固定 "sports" |
| status              | string       | 否       | active / resolved |
| end_time            | int64        | 否       | 结束时间戳（毫秒） |
| platform_count      | int          | 否       | 可用平台数 |
| volume              | float64      | 否       | 交易量 |
| save_pct            | float64      | 否       | 最优价比参考价节省百分比；(最高价−最低价)/最低价×100 |
| best_price_platform | string       | 否       | 最优价平台名 |
| outcomes             | []OutcomeItem| 是       | YES/NO 百分比 |
| event_uuid          | string       | 否       | 首平台 event_uuid |

#### OutcomeItem 子结构

| 参数名 | 字段类型 | 是否可空 | 备注 |
| ------ | -------- | -------- | ---- |
| label  | string   | 否       | YES/NO |
| price  | float64  | 否       | 0-1 概率 |
| pct    | int      | 否       | 0-100 百分比 |

#### 请求样例

```
GET http://localhost:8081/api/markets?type=sports&status=active&page=1&page_size=20
```

#### 响应样例

```json
{
  "page": 1,
  "page_size": 20,
  "total": 200,
  "items": [
    {
      "canonical_id": 1,
      "title": "Lakers win NBA Championship 2026?",
      "description": "...",
      "type": "sports",
      "status": "active",
      "end_time": 1735689600,
      "platform_count": 2,
      "volume": 10000,
      "save_pct": 5.2,
      "best_price_platform": "Polymarket",
      "outcomes": [{"label": "YES", "price": 0.65, "pct": 65}, {"label": "NO", "price": 0.35, "pct": 35}],
      "event_uuid": "..."
    }
  ]
}
```

---

### 2. 市场详情与多平台赔率

市场详情与多平台赔率。

- **接口 path:** `GET /api/markets/:event_uuid`
- **接口协议:** HTTP GET

#### 接口请求参数

| 请求参数  | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------- | -------- | -------- | ------ | ---- |
| event_uuid| string   | 是       | -      | 赛事 UUID 或 canonical_id（数字） |

#### 接口响应参数

| 参数名           | 字段类型   | 是否可空 | 备注 |
| ---------------- | ---------- | -------- | ---- |
| event            | EventInfo  | 否       | 赛事基本信息 |
| platform_options | []PlatformOption | 是 | 各平台选项与赔率 |
| analytics        | Analytics  | 否       | 汇总统计 |

#### EventInfo 子结构

| 参数名     | 字段类型 | 是否可空 | 备注 |
| ---------- | -------- | -------- | ---- |
| event_uuid | string   | 否       | 赛事 UUID |
| title      | string   | 否       | 赛事标题 |
| type       | string   | 否       | 类型 |
| status     | string   | 否       | 状态 |
| start_time | int64    | 否       | 开始时间戳（秒） |
| end_time   | int64    | 否       | 结束时间戳（秒） |

#### PlatformOption 子结构

| 参数名       | 字段类型 | 是否可空 | 备注 |
| ------------ | -------- | -------- | ---- |
| platform_id  | int      | 否       | 平台 ID |
| platform_name| string   | 否       | 平台名称 |
| option_name  | string   | 否       | 选项名，如 YES/NO |
| price        | float64  | 否       | 赔率（0~1） |

#### Analytics 子结构

| 参数名               | 字段类型 | 是否可空 | 备注 |
| -------------------- | -------- | -------- | ---- |
| best_price           | float64  | 否       | 最高赔率 |
| best_price_platform  | string   | 否       | 最高赔率所在平台 |
| best_price_option    | string   | 否       | 最高赔率对应选项 |
| platform_count       | int      | 否       | 平台数 |
| option_count         | int      | 否       | 选项总数 |
| volume               | float64  | 否       | 交易量 |
| price_min            | float64  | 否       | 最低赔率 |
| price_max            | float64  | 否       | 最高赔率 |
| price_spread_pct     | float64  | 否       | 价差百分比 (max-min)/max×100 |

#### 请求样例

```
GET http://localhost:8081/api/markets/evt-xxx
```

#### 响应样例

```json
{
  "event": {
    "event_uuid": "...",
    "title": "...",
    "type": "...",
    "status": "active",
    "start_time": 1735603200,
    "end_time": 1735689600
  },
  "platform_options": [
    {"platform_id": 1, "platform_name": "Polymarket", "option_name": "YES", "price": 0.65},
    {"platform_id": 1, "platform_name": "Polymarket", "option_name": "NO", "price": 0.35}
  ],
  "analytics": {
    "best_price": 0.65,
    "best_price_platform": "Polymarket",
    "best_price_option": "YES",
    "platform_count": 2,
    "option_count": 4,
    "volume": 10000,
    "price_min": 0.35,
    "price_max": 0.65,
    "price_spread_pct": 46.2
  }
}
```

---

## 订单

### 3. 下单准备（获取待签名信息）

后端**实时向三方平台查询赔率**并选出当前最高赔率，返回锁定赔率与待签名消息；用户对 `message_to_sign` 做 personal_sign 后再调用 **POST /api/orders/place** 并带上签名。

- **接口 path:** `POST /api/orders/prepare`
- **接口协议:** HTTP POST

#### 接口请求参数

| 请求参数        | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------------- | -------- | -------- | ------ | ---- |
| contract_order_id | string | 是       | -      | 入金后得到的合约订单号（betId 十六进制） |
| event_uuid      | string   | 是       | -      | 赛事 event_uuid 或 canonical_id |
| bet_option      | string   | 是       | -      | 下注方向，如 YES / NO |

#### 接口响应参数

| 参数名           | 字段类型 | 是否可空 | 备注 |
| ---------------- | -------- | -------- | ---- |
| locked_odds      | float64  | 否       | 当前实时最高赔率（0~1） |
| message_to_sign  | string   | 否       | 用户需 personal_sign 的原文，约 5 分钟有效 |
| expires_at_sec   | int64    | 否       | 过期时间戳（秒） |

#### 请求样例

```json
POST http://localhost:8081/api/orders/prepare
Content-Type: application/json

{
  "contract_order_id": "abc123...",
  "event_uuid": "evt-uuid-or-canonical-id",
  "bet_option": "YES"
}
```

#### 响应样例

```json
{
  "locked_odds": 0.65,
  "message_to_sign": "PlaceOrder:abc123:evt-uuid:YES:0.650000:1735689900",
  "expires_at_sec": 1735689900
}
```

**Error:** 400 — 缺少参数、未找到对应入账事件、或无可用赔率等，body 为 `{"error": "..."}`。

---

### 4. 下单

下单。可选带 `message_to_sign` + `signature`；若携带则先校验签名再按实时赔率选平台下单，不向前端暴露具体平台。

- **接口 path:** `POST /api/orders/place`
- **接口协议:** HTTP POST

#### 接口请求参数

| 请求参数        | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------------- | -------- | -------- | ------ | ---- |
| contract_order_id | string | 是       | -      | 入金得到的合约订单号 |
| event_uuid      | string   | 是       | -      | 赛事 event_uuid 或 canonical_id |
| bet_option      | string   | 是       | -      | 下注方向，如 YES / NO |
| amount          | float64  | 否       | -      | 下注金额，用于与入账金额校验 |
| message_to_sign | string   | 否       | -      | prepare 返回的待签名消息（与 signature 成对） |
| signature       | string   | 否       | -      | 对 message_to_sign 的 personal_sign 结果 |

#### 接口响应参数

| 参数名           | 字段类型 | 是否可空 | 备注 |
| ---------------- | -------- | -------- | ---- |
| order_uuid       | string   | 否       | 订单 UUID（与 contract_order_id 一致） |
| platform_order_id| string   | 否       | 三方平台订单号 |
| platform_id      | int      | 否       | 实际下单的平台 ID |
| status           | string   | 否       | 订单状态，如 placed |

#### 请求样例

```json
POST http://localhost:8081/api/orders/place
Content-Type: application/json

{
  "contract_order_id": "abc123...",
  "event_uuid": "evt-uuid-or-canonical-id",
  "bet_option": "YES",
  "amount": 10.5,
  "message_to_sign": "PlaceOrder:...",
  "signature": "0x..."
}
```

#### 响应样例

```json
{
  "order_uuid": "...",
  "platform_order_id": "...",
  "platform_id": 1,
  "status": "placed"
}
```

---

### 5. 订单列表

订单列表，按钱包与状态筛选、分页。

- **接口 path:** `GET /api/orders`
- **接口协议:** HTTP GET

#### 接口请求参数

| 请求参数  | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------- | -------- | -------- | ------ | ---- |
| wallet    | string   | 是       | -      | 用户钱包地址（0x...） |
| status    | string   | 否       | -      | 筛选状态，如 settled 表示可提现订单 |
| page      | int      | 否       | 1      | 当前查询页数 |
| page_size | int      | 否       | 20     | 每页返回的记录数 |

#### 接口响应参数

| 参数名 | 字段类型   | 是否可空 | 默认值 | 备注 |
| ------ | ---------- | -------- | ------ | ---- |
| page   | int        | 否       | 0      | 当前页数 |
| page_size | int     | 否       | 0      | 当前页返回的记录数 |
| total  | int64      | 否       | 0      | 总符合条件数 |
| items  | []OrderItem| 是       | 空     | 订单列表 |

#### OrderItem 子结构（列表项）

| 参数名             | 字段类型 | 是否可空 | 备注 |
| ------------------ | -------- | -------- | ---- |
| order_uuid         | string   | 否       | 订单 UUID |
| user_wallet        | string   | 否       | 用户钱包地址 |
| event_title        | string   | 否       | 赛事标题 |
| event_id           | int64    | 否       | 赛事 ID |
| platform_id        | int      | 否       | 平台 ID |
| platform_order_id  | string   | 是       | 三方平台订单号（可选） |
| bet_option         | string   | 否       | 下注方向 YES/NO |
| bet_amount         | float64  | 否       | 下注金额 |
| locked_odds        | float64  | 否       | 锁定赔率 |
| status             | string   | 否       | 订单状态 |
| created_at         | int64    | 否       | 创建时间戳（毫秒） |

#### 请求样例

```
GET http://localhost:8081/api/orders?wallet=0x...&status=settled&page=1&page_size=20
```

#### 响应样例

```json
{
  "page": 1,
  "page_size": 20,
  "total": 10,
  "items": [
    {
      "order_uuid": "...",
      "user_wallet": "0x...",
      "event_title": "...",
      "event_id": 1,
      "platform_id": 1,
      "platform_order_id": "...",
      "bet_option": "YES",
      "bet_amount": 10,
      "locked_odds": 0.65,
      "status": "placed",
      "created_at": 1735689600
    }
  ]
}
```

---

### 6. 订单详情

订单详情。

- **接口 path:** `GET /api/orders/:order_uuid`
- **接口协议:** HTTP GET

#### 接口请求参数

| 请求参数  | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------- | -------- | -------- | ------ | ---- |
| order_uuid| string   | 是       | -      | 订单 UUID（与 contract_order_id 一致） |

#### 接口响应参数

| 参数名              | 字段类型 | 是否可空 | 备注 |
| ------------------- | -------- | -------- | ---- |
| order_uuid          | string   | 否       | 订单 UUID |
| platform_order_id   | string   | 否       | 三方平台订单号 |
| user_wallet         | string   | 否       | 用户钱包 |
| event_id            | int64    | 否       | 赛事 ID |
| event_uuid          | string   | 否       | 赛事 UUID |
| event_title         | string   | 否       | 赛事标题 |
| platform_id         | int      | 否       | 平台 ID |
| bet_option          | string   | 否       | 下注方向 YES/NO |
| bet_amount          | float64  | 否       | 下注金额 |
| fund_currency       | string   | 否       | 资金币种，如 USDC |
| locked_odds         | float64  | 否       | 锁定赔率 |
| expected_profit     | float64  | 否       | 预期利润 |
| actual_profit       | float64  | 否       | 实际利润 |
| status              | string   | 否       | placed / settled / withdrawn 等 |
| fund_lock_tx_hash   | string   | 是       | 入金交易哈希（可选） |
| settlement_tx_hash  | string   | 是       | 结算交易哈希（可选） |
| start_time          | int64    | 否       | 盘口开始时间（毫秒） |
| end_time            | int64    | 否       | 盘口结束时间（毫秒） |
| created_at          | int64    | 否       | 创建时间（毫秒） |
| updated_at          | int64    | 否       | 更新时间（毫秒） |

#### 请求样例

```
GET http://localhost:8081/api/orders/order-uuid-xxx
```

#### 响应样例

```json
{
  "order_uuid": "...",
  "platform_order_id": "...",
  "user_wallet": "0x...",
  "event_id": 1,
  "event_uuid": "...",
  "event_title": "...",
  "platform_id": 1,
  "bet_option": "YES",
  "bet_amount": 10,
  "fund_currency": "USDC",
  "locked_odds": 0.65,
  "expected_profit": 1.5,
  "actual_profit": 1.2,
  "status": "settled",
  "fund_lock_tx_hash": "0x...",
  "settlement_tx_hash": "0x...",
  "start_time": 1735603200000,
  "end_time": 1735689600000,
  "created_at": 1735689600000,
  "updated_at": 1735690000000
}
```

---

### 7. 获取提现参数

获取提现参数。仅当订单 `status` 为 `settled` 时可调用。

- **接口 path:** `GET /api/orders/:order_uuid/withdraw-info`
- **接口协议:** HTTP GET

#### 接口请求参数

| 请求参数  | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------- | -------- | -------- | ------ | ---- |
| order_uuid| string   | 是       | -      | 订单 UUID |

#### 接口响应参数

根据订单类型返回不同结构：Kalshi 订单（`type: "kalshi"`）由后端处理兑付；链上订单（`type: "chain"`）需用户签名并支付 Gas。

| 参数名           | 字段类型 | 是否可空 | 备注 |
| ---------------- | -------- | -------- | ---- |
| order_uuid       | string   | 否       | 订单 UUID |
| user_wallet      | string   | 否       | 用户钱包地址 |
| type             | string   | 否       | kalshi：后端处理；chain：链上用户签名 |
| amount           | float64  | 否       | 可提金额 |
| fee              | float64  | 是       | 1% 手续费（仅 Kalshi） |
| user_amount      | float64  | 是       | 用户实得（仅 Kalshi） |
| contract_address | string   | 是       | 提现合约地址（仅 chain） |
| method           | string   | 是       | 合约方法名，如 withdraw（仅 chain） |
| message          | string   | 否       | 提示文案 |

#### 请求样例

```
GET http://localhost:8081/api/orders/order-uuid-xxx/withdraw-info
```

#### 响应样例（Kalshi）

```json
{
  "order_uuid": "...",
  "user_wallet": "0x...",
  "type": "kalshi",
  "amount": 11.2,
  "fee": 0.1,
  "user_amount": 11.1,
  "contract_address": "",
  "method": "",
  "message": "后端将处理提现（Circle USD→USDC，1% 手续费入 FeeVault）"
}
```

#### 响应样例（链上）

```json
{
  "order_uuid": "...",
  "user_wallet": "0x...",
  "type": "chain",
  "amount": 11.2,
  "contract_address": "0x...",
  "method": "withdraw",
  "message": "用户签名并支付 Gas 完成链上提现，Gas 费由用户承担"
}
```

---

### 8. 发起提现

发起提现。仅当订单 `status` 为 `settled` 时可调用。Kalshi：后端执行兑付与链上转账，订单状态更新为 `withdrawn`。链上：仅记录请求；实际提现由前端根据 withdraw-info 调合约、用户签名完成。

- **接口 path:** `POST /api/orders/:order_uuid/withdraw`
- **接口协议:** HTTP POST

#### 接口请求参数

| 请求参数  | 请求类型 | 是否必填 | 默认值 | 备注 |
| --------- | -------- | -------- | ------ | ---- |
| order_uuid| string   | 是       | -      | 订单 UUID（Path） |

#### 接口响应参数

| 参数名  | 字段类型 | 是否可空 | 备注 |
| ------- | -------- | -------- | ---- |
| message | string   | 否       | 成功时的提示文案 |

#### 请求样例

```
POST http://localhost:8081/api/orders/order-uuid-xxx/withdraw
```

#### 响应样例

```json
{
  "message": "提现请求已记录"
}
```

**Error:** 400 — 订单状态不是 `settled`，body 为 `{"error": "..."}`。

---

## 同步（内部/运维）

### 9. 触发平台事件同步

触发指定平台事件同步。

- **接口 path:** `POST /sync/platform/:platform`
- **接口协议:** HTTP POST

#### 接口请求参数

| 请求参数 | 请求类型 | 是否必填 | 默认值 | 备注 |
| -------- | -------- | -------- | ------ | ---- |
| platform | string   | 是       | -      | 平台标识：polymarket 或 kalshi（Path） |

#### 接口响应

- 200：同步已触发或执行完成，具体响应体以实际实现为准。

#### 请求样例

```
POST http://localhost:8081/sync/platform/polymarket
```
