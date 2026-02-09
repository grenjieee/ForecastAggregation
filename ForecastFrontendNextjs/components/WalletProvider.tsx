"use client";
import { WagmiProvider } from "wagmi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RainbowKitProvider, darkTheme } from "@rainbow-me/rainbowkit";
import { config } from "@/lib/wagmi";
import { ReactNode } from "react"; // 添加导入
interface WalletProviderProps {
  children: ReactNode;
}
function WalletProvider({ children }: WalletProviderProps) {
  const queryClient = new QueryClient({});
  return <WagmiProvider config={config}>
    <QueryClientProvider client={queryClient}>
      <RainbowKitProvider
        locale="en-US"
        theme={darkTheme({
          accentColor: "#C7FC6A",
          accentColorForeground: "#000",
        })}
      >{children}</RainbowKitProvider>
    </QueryClientProvider>
  </WagmiProvider>
}


export default WalletProvider;
