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

export interface Market {
  id: string;
  title: string;
  description: string;
  category: 'Politics' | 'Sports' | 'Crypto' | 'Finance' | 'Tech' | 'Culture';
  endDate: string;
  totalVolume: string;
  platforms: PlatformPrice[];
  imageUrl?: string;
}

export const mockMarkets: Market[] = [
  {
    id: '1',
    title: 'Bitcoin price above $70,000 on Feb 10, 2026?',
    description: 'Will Bitcoin (BTC) close above $70,000 USD on February 10, 2026 at 11:59 PM UTC?',
    category: 'Crypto',
    endDate: '2026-02-10T23:59:59Z',
    totalVolume: '$2.4M',
    platforms: [
      {
        platform: 'Polymarket',
        yesPrice: 0.34,
        noPrice: 0.66,
        volume: '$1.2M',
        liquidity: '$450K',
        fee: '2%',
        url: 'https://polymarket.com'
      },
      {
        platform: 'Kalshi',
        yesPrice: 0.31,
        noPrice: 0.69,
        volume: '$850K',
        liquidity: '$320K',
        fee: '3%',
        url: 'https://kalshi.com'
      },
      {
        platform: 'Robinhood',
        yesPrice: 0.36,
        noPrice: 0.64,
        volume: '$350K',
        liquidity: '$180K',
        fee: '1.5%',
        url: 'https://robinhood.com'
      }
    ]
  },
  {
    id: '2',
    title: 'Trump wins 2028 Republican nomination?',
    description: 'Will Donald Trump be the Republican nominee for the 2028 US Presidential election?',
    category: 'Politics',
    endDate: '2028-08-15T23:59:59Z',
    totalVolume: '$5.8M',
    platforms: [
      {
        platform: 'Polymarket',
        yesPrice: 0.42,
        noPrice: 0.58,
        volume: '$3.2M',
        liquidity: '$890K',
        fee: '2%',
        url: 'https://polymarket.com'
      },
      {
        platform: 'Kalshi',
        yesPrice: 0.45,
        noPrice: 0.55,
        volume: '$1.8M',
        liquidity: '$620K',
        fee: '3%',
        url: 'https://kalshi.com'
      },
      {
        platform: 'Robinhood',
        yesPrice: 0.40,
        noPrice: 0.60,
        volume: '$800K',
        liquidity: '$340K',
        fee: '1.5%',
        url: 'https://robinhood.com'
      }
    ]
  },
  {
    id: '3',
    title: 'Lakers win NBA Championship 2026?',
    description: 'Will the Los Angeles Lakers win the 2025-2026 NBA Championship?',
    category: 'Sports',
    endDate: '2026-06-30T23:59:59Z',
    totalVolume: '$1.9M',
    platforms: [
      {
        platform: 'Polymarket',
        yesPrice: 0.18,
        noPrice: 0.82,
        volume: '$950K',
        liquidity: '$380K',
        fee: '2%',
        url: 'https://polymarket.com'
      },
      {
        platform: 'Kalshi',
        yesPrice: 0.16,
        noPrice: 0.84,
        volume: '$620K',
        liquidity: '$240K',
        fee: '3%',
        url: 'https://kalshi.com'
      },
      {
        platform: 'Robinhood',
        yesPrice: 0.20,
        noPrice: 0.80,
        volume: '$330K',
        liquidity: '$150K',
        fee: '1.5%',
        url: 'https://robinhood.com'
      }
    ]
  },
  {
    id: '4',
    title: 'Ethereum reaches $5,000 before 2027?',
    description: 'Will Ethereum (ETH) reach or exceed $5,000 USD at any point before January 1, 2027?',
    category: 'Crypto',
    endDate: '2026-12-31T23:59:59Z',
    totalVolume: '$3.2M',
    platforms: [
      {
        platform: 'Polymarket',
        yesPrice: 0.56,
        noPrice: 0.44,
        volume: '$1.8M',
        liquidity: '$640K',
        fee: '2%',
        url: 'https://polymarket.com'
      },
      {
        platform: 'Kalshi',
        yesPrice: 0.53,
        noPrice: 0.47,
        volume: '$980K',
        liquidity: '$420K',
        fee: '3%',
        url: 'https://kalshi.com'
      },
      {
        platform: 'Robinhood',
        yesPrice: 0.58,
        noPrice: 0.42,
        volume: '$420K',
        liquidity: '$210K',
        fee: '1.5%',
        url: 'https://robinhood.com'
      }
    ]
  },
  {
    id: '5',
    title: 'Fed cuts rates by March 2026?',
    description: 'Will the Federal Reserve cut interest rates by at least 25 basis points before March 31, 2026?',
    category: 'Finance',
    endDate: '2026-03-31T23:59:59Z',
    totalVolume: '$4.5M',
    platforms: [
      {
        platform: 'Polymarket',
        yesPrice: 0.72,
        noPrice: 0.28,
        volume: '$2.4M',
        liquidity: '$780K',
        fee: '2%',
        url: 'https://polymarket.com'
      },
      {
        platform: 'Kalshi',
        yesPrice: 0.68,
        noPrice: 0.32,
        volume: '$1.5M',
        liquidity: '$560K',
        fee: '3%',
        url: 'https://kalshi.com'
      },
      {
        platform: 'Robinhood',
        yesPrice: 0.74,
        noPrice: 0.26,
        volume: '$600K',
        liquidity: '$290K',
        fee: '1.5%',
        url: 'https://robinhood.com'
      }
    ]
  },
  {
    id: '6',
    title: 'Apple releases AR glasses in 2026?',
    description: 'Will Apple officially announce and release consumer AR/VR glasses in 2026?',
    category: 'Tech',
    endDate: '2026-12-31T23:59:59Z',
    totalVolume: '$1.6M',
    platforms: [
      {
        platform: 'Polymarket',
        yesPrice: 0.28,
        noPrice: 0.72,
        volume: '$820K',
        liquidity: '$340K',
        fee: '2%',
        url: 'https://polymarket.com'
      },
      {
        platform: 'Kalshi',
        yesPrice: 0.25,
        noPrice: 0.75,
        volume: '$540K',
        liquidity: '$230K',
        fee: '3%',
        url: 'https://kalshi.com'
      },
      {
        platform: 'Robinhood',
        yesPrice: 0.30,
        noPrice: 0.70,
        volume: '$240K',
        liquidity: '$120K',
        fee: '1.5%',
        url: 'https://robinhood.com'
      }
    ]
  }
];

export const categories = ['All', 'Politics', 'Sports', 'Crypto', 'Finance', 'Tech', 'Culture'] as const;

export function getBestPrice(market: Market): { platform: string; yesPrice: number; savings: string } {
  const bestYes = market.platforms.reduce((best, current) => 
    current.yesPrice < best.yesPrice ? current : best
  );
  
  const worstYes = market.platforms.reduce((worst, current) => 
    current.yesPrice > worst.yesPrice ? current : worst
  );
  
  const savings = ((worstYes.yesPrice - bestYes.yesPrice) / worstYes.yesPrice * 100).toFixed(1);
  
  return {
    platform: bestYes.platform,
    yesPrice: bestYes.yesPrice,
    savings: `${savings}%`
  };
}
