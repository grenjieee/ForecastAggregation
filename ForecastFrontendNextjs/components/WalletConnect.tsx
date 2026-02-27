/**
 * Design: Cyberpunk Neon Futurism
 * - Neon green glow effect for connected state
 * - Pulse animation to attract attention
 * - Monospace font for wallet address
 */

import { Button } from "@/components/ui/button";
import { Wallet, Copy, LogOut, ChevronDown, Check } from "lucide-react"; // Added Check icon
import { toast } from "sonner";
import ConnectWallet from "./ConnectWallet";
import { useAccount, useDisconnect } from "wagmi";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useState } from "react";

export function WalletConnect() {
  const { address, isConnected } = useAccount();
  const { disconnect } = useDisconnect();
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    if (address) {
      navigator.clipboard.writeText(address);
      setCopied(true);
      toast.success("Address copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleDisconnect = () => {
    disconnect();
    toast.info("Wallet disconnected");
  };

  return isConnected ? (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          className="neon-border-gradient neon-glow-green hover:neon-glow-green hover:scale-105 transition-all duration-200 mono-font group"
        >
          <Wallet className="mr-2 h-4 w-4" />
          {address ? `${address.substring(0, 6)}...${address.substring(address.length - 4)}` : ''}
          <ChevronDown className="ml-2 h-4 w-4 opacity-50 group-hover:opacity-100 transition-opacity" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56 bg-black/90 border-[#00ff9d]/30 text-[#00ff9d]">
        <DropdownMenuLabel>My Wallet</DropdownMenuLabel>
        <DropdownMenuSeparator className="bg-[#00ff9d]/20" />
        <DropdownMenuItem 
          onClick={handleCopy}
          className="cursor-pointer focus:bg-[#00ff9d]/10 focus:text-[#00ff9d] transition-colors"
        >
          {copied ? <Check className="mr-2 h-4 w-4" /> : <Copy className="mr-2 h-4 w-4" />}
          <span>{copied ? "Copied!" : "Copy Address"}</span>
        </DropdownMenuItem>
        <DropdownMenuItem 
          onClick={handleDisconnect}
          className="cursor-pointer focus:bg-red-500/10 focus:text-red-500 text-red-400 transition-colors"
        >
          <LogOut className="mr-2 h-4 w-4" />
          <span>Disconnect</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ) : <ConnectWallet />;
}
