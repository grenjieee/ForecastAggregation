import { MarketQueryResult } from "../mockData";

const BASE_URL = "http://47.86.169.161/api";

export const fetchMarkets = async (
  type: string,
  status: string,
  page: number = 1,
  pageSize: number = 20
): Promise<MarketQueryResult> => {
  const response = await fetch(
    `${BASE_URL}/markets?type=${type}&status=${status}&page=${page}&page_size=${pageSize}`
  );

  if (!response.ok) {
    throw new Error("Failed to fetch markets");
  }
  console.log("Fetched markets:", await response.clone().json());
  return response.json();
};