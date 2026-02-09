/**
 * Design: Cyberpunk Neon Futurism
 * - Neon green glow effect for connected state
 * - Pulse animation to attract attention
 * - Monospace font for wallet address
 */

import { Button } from "@/components/ui/button";
import { Wallet } from "lucide-react";
import { toast } from "sonner";
import ConnectWallet from "./ConnectWallet";
import { useAccount } from "wagmi";

export function WalletConnect() {
  const { address, isConnected } = useAccount();
  // const connectWallet = async () => {
  //   // Simulate wallet connection
  //   toast.info("Connecting wallet...", {
  //     description: "This is a demo. In production, this would connect to MetaMask or WalletConnect."
  //   });

  //   // Simulate delay
  //   setTimeout(() => {
  //     const mockAddress = "0x" + Math.random().toString(16).substring(2, 42);
  //     setAddress(mockAddress);
  //     setIsConnected(true);
  //     toast.success("Wallet connected!", {
  //       description: `Connected to ${mockAddress.substring(0, 6)}...${mockAddress.substring(38)}`
  //     });
  //   }, 1000);
  // };

  const disconnectWallet = () => {
    toast.info("Wallet disconnected");
  };

  return isConnected ? (
    <Button
      onClick={disconnectWallet}
      variant="outline"
      className="neon-border-gradient neon-glow-green hover:neon-glow-green hover:scale-105 transition-all duration-200 mono-font"
    >
      <Wallet className="mr-2 h-4 w-4" />
      {address ? `${address.substring(0, 6)}...${address.substring(38)}` : ''}
    </Button>
  ) : <ConnectWallet />;


}
