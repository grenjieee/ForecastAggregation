export const chainIdList = ["84532", "97"];

let chain: string = chainIdList[0];

export const network = "testnet";
export const chainObj: any = {
  "84532": {
    apiUrl: "https://api.xais.fun",
    baseScanUrl: "https://sepolia.basescan.org",
    networkName: "Base Sepolia Testnet",
    addresses: {
      ["84532"]: {
        memeFactoryAddress: "0xfFb0934487D4E3d72d21a4634ba7ccE28e84c07E",
      },
    },
    ROUTER_ADDRESS: "0x1689E7B1F10000AE47eBfE339a4f69dECd19F602",
    transferAddress: "0xc825055a970bd50f7afed5d030e3e82d89b39eb7",
    graphKey: "Bearer bd7d231f4f716283d32e9623a3bf4625",
    graphUrl: `https://api.studio.thegraph.com/query/104859/xai/version/latest`,
    geckoterminalPool: "base",
    toChainUrl: "https://lang.xais.fun/home",
    cdnUrl: "https://cdn.xais.fun",
    mainCoinName: "ETH",
    okLinkName: "BASE",
    initialCoinAmount: 3,
    toUniswapAmount: 6,
    toTopSpotAmount: 4,
    toUniswapMarketCap: 22.5,
    stableCoinName: "USDC",
    stableCoinAddress: "0x0d807205e030ef21cEEfB16736a194Cd95Fb3748",
    stableCoinDecimals: "18",
  },
  "97": {
    apiUrl: "https://bsc.xais.fun",
    baseScanUrl: "https://testnet.bscscan.com",
    networkName: "BNB Smart Chain Testnet",
    addresses: {
      ["97"]: {
        memeFactoryAddress: "0x7fD7E87F967497103f46dCb57328e5243a30094d",
      },
    },
    ROUTER_ADDRESS: "0x823A57b4B63971BFf7E3B050acaE71E2A2123Ce6",
    transferAddress: "0x22c28f307834f22b7111e13a884e2a2c0576dc2a",
    graphKey: "",
    graphUrl: `http://150.5.175.171:8000/subgraphs/name/xai_test/graphql`,
    geckoterminalPool: "bsc",
    toChainUrl: "https://lang.xais.fun/home",
    cdnUrl: "https://cdnbn.xais.fun",
    mainCoinName: "BNB",
    okLinkName: "BSC",
    initialCoinAmount: 3 * 5,
    toUniswapAmount: 6 * 5,
    toTopSpotAmount: 4 * 5,
    toUniswapMarketCap: 112.5,
    stableCoinName: "USDT",
    stableCoinAddress: "",
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
