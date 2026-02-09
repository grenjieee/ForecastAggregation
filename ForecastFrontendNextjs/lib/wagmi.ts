import { getDefaultConfig } from "@rainbow-me/rainbowkit";
const walletConnectProjectId = "6725398ec65294a699e578a584b4bdd3";
import { WagmiProvider, http, custom } from "wagmi";
import {
  mainnet,
  sepolia,
  base,
  baseSepolia,
  opBNBTestnet,
  bsc,
  bscTestnet,
} from "@wagmi/core/chains";
import { connectorsForWallets } from "@rainbow-me/rainbowkit";
import {
  rainbowWallet,
  walletConnectWallet,
  okxWallet,
  metaMaskWallet,
  coinbaseWallet,
  bitgetWallet,
  gateWallet,
  binanceWallet,
} from "@rainbow-me/rainbowkit/wallets";
import { chainIdList, network } from "@/lib/constants";
import { createConfig } from "wagmi";
const chain: string = chainIdList[0];
const connectors = connectorsForWallets(
  [
    {
      groupName: "Recommended",
      wallets: [
        metaMaskWallet,
        binanceWallet,
        okxWallet,
        gateWallet,
        // rainbowWallet,
        walletConnectWallet,
        bitgetWallet,
        coinbaseWallet,
      ],
    },
  ],
  {
    appName: "XAIS",
    projectId: walletConnectProjectId,
  }
);
// export const config = createConfig({
//   connectors,
//   // projectId: walletConnectProjectId,
//   chains: [base],
//   ssr: true, // If your dApp uses server side rendering (SSR)
// });
const customBaseSepolia = {
  ...baseSepolia,
  name: "Base Sepolia Testnet",
  rpcUrls: {
    default: {
      http: ["https://base-sepolia-rpc.publicnode.com"], // 这里是新的 RPC URL
    },
  },
};
const customBscTestnet = {
  ...bscTestnet,
  name: "BNB Smart Chain Testnet",
};
//  https://public-bsc.nownodes.io
const customBsc = {
  ...bsc,
  name: "BNB Smart Chain Mainnet",
  rpcUrls: {
    default: { http: ["https://public-bsc.nownodes.io"] },
  },
};
const chainObj: any =
  network === "testnet"
    ? {
      "84532": {
        chain: customBaseSepolia,
      },
      "97": {
        chain: customBscTestnet,
      },
    }
    : {
      "8453": {
        chain: base,
      },
      "56": {
        chain: customBsc,
      },
    };
export const mChain = chainObj[chain].chain;
// const customBaseSepolia = baseSepolia;
export const config = createConfig({
  connectors,
  ssr: true,
  chains: [mChain],
  transports: {
    [mChain.id]: http(mChain.rpcUrls.default.http[0]),
  },
});
// export const config = getDefaultConfig({
//   projectId: walletConnectProjectId,
//   chains: [base],
//   ssr: true, // If your dApp uses server side rendering (SSR)
// });

// export const config = createConfig({
//   autoConnect: true,
//   connectors,
//   publicClient,
//   webSocketPublicClient,
// });
