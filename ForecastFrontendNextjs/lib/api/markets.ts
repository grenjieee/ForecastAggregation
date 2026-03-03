import { MarketQueryResult } from "../mockData";

const BASE_URL = "http://47.86.169.161/api";

// 获取市场列表
export const fetchMarkets = async (
  type: string,
  status: string,
  page: number = 1,
  pageSize: number = 20
): Promise<MarketQueryResult> => {
  const response = await fetch(
    `${BASE_URL}/markets?type=${encodeURIComponent(type)}&status=${encodeURIComponent(
      status
    )}&page=${page}&page_size=${pageSize}`
  );

  if (!response.ok) {
    throw new Error("Failed to fetch markets");
  }

  console.log("Fetched markets:", await response.clone().json());
  return response.json();
};

// 获取市场详情
export const fetchMarketDetail = async (eventUuid: string): Promise<MarketDetail> => {
  try {
    const response = await fetch(
      `${BASE_URL}/markets/${encodeURIComponent(eventUuid)}`
    );

    if (!response.ok) {
      throw new Error(`Failed to fetch market detail: ${response.statusText}`);
    }

    const data: MarketDetail = await response.json(); // 解析返回的 JSON 数据
    console.log("Market Detail:", data); // 打印返回的数据
    return data; // 返回数据
  } catch (error) {
    console.error("Error fetching market detail:", error);
    throw error; // 抛出错误以便调用方处理
  }
};

export interface MarketEvent {
  event_uuid: string;
  title: string;
  type: string;
  status: string;
  start_time: number;
  end_time: number;
}

export interface PlatformOption {
  platform_id: number;
  platform_name: string;
  option_name: string;
  price: number;
}

export interface Analytics {
  best_price: number;
  best_price_platform: string;
  best_price_option: string;
  platform_count: number;
  option_count: number;
  volume: number;
  price_min: number;
  price_max: number;
  price_spread_pct: number;
}

export interface MarketDetail {
  event: MarketEvent;
  platform_options: PlatformOption[];
  analytics: Analytics;
}

// 下单接口
export const placeOrder = async (payload: PlaceOrderPayload) => {
  const response = await fetch(`${BASE_URL}/orders/place`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error("Failed to place order");
  }

  return response.json();
};

// 获取订单列表
export const fetchOrders = async ({
  wallet,
  status,
  page = 1,
  pageSize = 20,
}: FetchOrdersParams) => {
  const query = new URLSearchParams();
  query.set("wallet", wallet);
  if (status) query.set("status", status);
  query.set("page", String(page));
  query.set("page_size", String(pageSize));

  const response = await fetch(`${BASE_URL}/orders?${query.toString()}`);

  if (!response.ok) {
    throw new Error("Failed to fetch orders");
  }

  return response.json();
};

// 获取订单详情
export const fetchOrderDetail = async (orderUuid: string) => {
  const response = await fetch(`${BASE_URL}/orders/${encodeURIComponent(orderUuid)}`);

  if (!response.ok) {
    throw new Error("Failed to fetch order detail");
  }

  return response.json();
};

// 获取提现信息
export const fetchWithdrawInfo = async (orderUuid: string) => {
  const response = await fetch(
    `${BASE_URL}/orders/${encodeURIComponent(orderUuid)}/withdraw-info`
  );

  if (!response.ok) {
    throw new Error("Failed to fetch withdraw info");
  }

  return response.json();
};

// 提现接口
export const withdrawOrder = async (
  orderUuid: string,
  payload?: { tx_hash?: string }
) => {
  const response = await fetch(
    `${BASE_URL}/orders/${encodeURIComponent(orderUuid)}/withdraw`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload ?? {}),
    }
  );

  if (!response.ok) {
    throw new Error("Failed to withdraw order");
  }

  return response.json();
};

export interface PlaceOrderPayload {
  contract_order_id: string;
  event_uuid: string;
  bet_option: string;
  amount?: number;
  wallet?: string;
}

export interface FetchOrdersParams {
  wallet: string;
  status?: string;
  page?: number;
  pageSize?: number;
}