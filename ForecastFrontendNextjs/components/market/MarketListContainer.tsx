import { fetchMarkets } from "@/lib/api/markets";
import { MarketList } from "./MarketList";
import { Market, MarketQueryResult } from "@/lib/mockData";


export const revalidate = 0; // 禁止缓存，确保每次请求都获取最新数据
export default async function MarketListContainer() {
    const initialData: MarketQueryResult = await fetchMarkets("sports", "active", 1, 20);
    return (
        <MarketList
            {...initialData}
        />
    );
}
