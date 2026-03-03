// Mock data for prediction markets aggregator

export interface PlatformPrice {
  platform: 'Polymarket' | 'Kalshi' | 'Robinhood';
  yesPrice: number;
  noPrice: number;
  volume: string;
  liquidity: string;
  fee: string;
  url: string;
}
export interface MarketQueryResult {
  page: number;
  page_size: number;
  total: number;
  items: Market[];
}
export interface Market {
  canonical_id: number;
  title: string;
  description: string;
  type: string; // Changed from 'category' to 'type'
  status: string;
  end_time: number; // Changed from 'endDate' to 'end_time' and type to number
  platform_count: number;
  volume: number; // Changed from 'totalVolume' to 'volume' and type to number
  save_pct: number;
  best_price_platform: string;
  outcomes: {
    label: string;
    price: number;
    pct: number;
  }[];
  event_uuid: string;
}

export const mockMarkets: Market[] = [
  {
    canonical_id: 1,
    title: 'Bitcoin price above $70,000 on Feb 10, 2026?',
    description: 'Will Bitcoin (BTC) close above $70,000 USD on February 10, 2026 at 11:59 PM UTC?',
    type: 'Crypto',
    status: 'active',
    end_time: 1754884799,
    platform_count: 3,
    volume: 2400000,
    save_pct: 0.34,
    best_price_platform: 'Polymarket',
    outcomes: [
      { label: 'Yes', price: 0.34, pct: 34 },
      { label: 'No', price: 0.66, pct: 66 },
    ],
    event_uuid: 'event-1',
  },
  {
    canonical_id: 2,
    title: 'Trump wins 2028 Republican nomination?',
    description: 'Will Donald Trump be the Republican nominee for the 2028 US Presidential election?',
    type: 'Politics',
    status: 'active',
    end_time: 1754884799,
    platform_count: 3,
    volume: 5800000,
    save_pct: 0.42,
    best_price_platform: 'Polymarket',
    outcomes: [
      { label: 'Yes', price: 0.42, pct: 42 },
      { label: 'No', price: 0.58, pct: 58 },
    ],
    event_uuid: 'event-2',
  },
  {
    canonical_id: 3,
    title: 'Lakers win NBA Championship 2026?',
    description: 'Will the Los Angeles Lakers win the 2025-2026 NBA Championship?',
    type: 'Sports',
    status: 'active',
    end_time: 1754884799,
    platform_count: 3,
    volume: 1900000,
    save_pct: 0.18,
    best_price_platform: 'Polymarket',
    outcomes: [
      { label: 'Yes', price: 0.18, pct: 18 },
      { label: 'No', price: 0.82, pct: 82 },
    ],
    event_uuid: 'event-3',
  },
  {
    canonical_id: 4,
    title: 'Ethereum reaches $5,000 before 2027?',
    description: 'Will Ethereum (ETH) reach or exceed $5,000 USD at any point before January 1, 2027?',
    type: 'Crypto',
    status: 'active',
    end_time: 1754884799,
    platform_count: 3,
    volume: 3200000,
    save_pct: 0.56,
    best_price_platform: 'Polymarket',
    outcomes: [
      { label: 'Yes', price: 0.56, pct: 56 },
      { label: 'No', price: 0.44, pct: 44 },
    ],
    event_uuid: 'event-4',
  },
  {
    canonical_id: 5,
    title: 'Fed cuts rates by March 2026?',
    description: 'Will the Federal Reserve cut interest rates by at least 25 basis points before March 31, 2026?',
    type: 'Finance',
    status: 'active',
    end_time: 1754884799,
    platform_count: 3,
    volume: 4500000,
    save_pct: 0.72,
    best_price_platform: 'Polymarket',
    outcomes: [
      { label: 'Yes', price: 0.72, pct: 72 },
      { label: 'No', price: 0.28, pct: 28 },
    ],
    event_uuid: 'event-5',
  },
  {
    canonical_id: 6,
    title: 'Apple releases AR glasses in 2026?',
    description: 'Will Apple officially announce and release consumer AR/VR glasses in 2026?',
    type: 'Tech',
    status: 'active',
    end_time: 1754884799,
    platform_count: 3,
    volume: 1600000,
    save_pct: 0.28,
    best_price_platform: 'Polymarket',
    outcomes: [
      { label: 'Yes', price: 0.28, pct: 28 },
      { label: 'No', price: 0.72, pct: 72 },
    ],
    event_uuid: 'event-6',
  }
];

export const categories = ['All', 'Politics', 'sports', 'Crypto', 'Finance', 'Tech', 'Culture'] as const;

export function getBestPrice(market: Market): { platform: string; yesPrice: number; savings: string } {
  const bestYes = market.outcomes.reduce((best, current) => 
    current.price < best.price ? current : best
  );
  
  const worstYes = market.outcomes.reduce((worst, current) => 
    current.price > worst.price ? current : worst
  );
  
  const savings = ((worstYes.price - bestYes.price) / worstYes.price * 100).toFixed(1);
  
  return {
    platform: bestYes.label,
    yesPrice: bestYes.price,
    savings: `${savings}%`
  };
}
