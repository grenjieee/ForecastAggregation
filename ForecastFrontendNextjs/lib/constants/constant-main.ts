export const chainIdList = ["8453", "56"];

let chain: string = chainIdList[0];


export const network = "mainnet";
export const chainObj: any = {
  "8453": {
    apiUrl: "https://base.xais.fun",
    baseScanUrl: "https://basescan.org",
    networkName: "Base",
    addresses: {
      ["8453"]: {
        memeFactoryAddress: "0x6C2Cc015d09c6172fCEb661A91152f10C7a1b513",
      },
    },
    ROUTER_ADDRESS: "0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24",
    transferAddress: "0x22c28f307834f22b7111e13a884e2a2c0576dc2a",
    graphKey: "Bearer bd7d231f4f716283d32e9623a3bf4625",
    graphUrl: `https://api.studio.thegraph.com/query/104859/xai_main/version/latest`,
    geckoterminalPool: "base",
    toChainUrl: "https://video.xais.fun/home",
    cdnUrl: "https://cdnbase.xais.fun",
    mainCoinName: "ETH",
    okLinkName: "BASE",
    initialCoinAmount: 3,
    toUniswapAmount: 6,
    toTopSpotAmount: 4,
    toUniswapMarketCap: 22.5,
    stableCoinName: "USDC",
    stableCoinAddress: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
    stableCoinDecimals: "6",
  },
  "56": {
    apiUrl: "https://bsc.xais.fun",
    baseScanUrl: "https://basescan.org/",
    networkName: "Binance Smart Chain",
    addresses: {
      ["56"]: {
        memeFactoryAddress: "0x810fA7ff0c4A17F2A3923db16D9589E98a11F2c7",
      },
    },
    ROUTER_ADDRESS: "0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24",
    transferAddress: "0x22c28f307834f22b7111e13a884e2a2c0576dc2a",
    graphKey: "Bearer bd7d231f4f716283d32e9623a3bf4625",
    graphUrl: `https://api.studio.thegraph.com/query/104859/bsc/version/latest`,
    toChainUrl: "https://video.xais.fun/home",
    geckoterminalPool: "bsc",
    cdnUrl: "https://cdnbn.xais.fun",
    mainCoinName: "BNB",
    okLinkName: "BSC",
    initialCoinAmount: 3 * 5,
    toUniswapAmount: 6 * 5,
    toTopSpotAmount: 4 * 5,
    toUniswapMarketCap: 112.5,
    stableCoinName: "USDT",
    stableCoinAddress: "0x55d398326f99059fF775485246999027B3197955",
    stableCoinDecimals: "18",
  },
};
export const apiUrl = chainObj[chain].apiUrl;
export const wssPath = apiUrl?.replace("https", "wss");

export const cdnUrl = chainObj[chain].cdnUrl;

export const baseScanUrl = chainObj[chain].baseScanUrl;
export const networkName = chainObj[chain].networkName;

export const marketInterval = 5000;
export const addresses: any = chainObj[chain].addresses;
export const ROUTER_ADDRESS = chainObj[chain].ROUTER_ADDRESS;
export const transferAddress = chainObj[chain].transferAddress;
export const graphKey = chainObj[chain].graphKey;
export const graphUrl = chainObj[chain].graphUrl;
export const geckoterminalPool = chainObj[chain].geckoterminalPool;
export const toChainUrl = chainObj[chain].toChainUrl;
export const mainCoinName = chainObj[chain].mainCoinName;
export const okLinkName = chainObj[chain].okLinkName;
export const initialCoinAmount = chainObj[chain].initialCoinAmount;
export const toUniswapAmount = chainObj[chain].toUniswapAmount;
export const toTopSpotAmount = chainObj[chain].toTopSpotAmount;
export const toUniswapMarketCap = chainObj[chain].toUniswapMarketCap;
export const stableCoinName = chainObj[chain].stableCoinName;
export const stableCoinAddress = chainObj[chain].stableCoinAddress;
export const stableCoinDecimals = chainObj[chain].stableCoinDecimals;

export const tonApiUrl = "https://testnet.tonapi.io";
