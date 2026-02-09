"use client";
import { useConnectModal } from "@rainbow-me/rainbowkit";
import { useAccount } from "wagmi";
import { Button } from "@/components/ui/button";
import { Wallet } from "lucide-react";

export default function ConnectWallet() {
  const { openConnectModal } = useConnectModal();
  const { address } = useAccount();

  return (
    <Button
      // onClick={connectWallet}
      className="neon-border-gradient bg-gradient-to-r from-[oklch(0.8_0.2_200)] to-[oklch(0.65_0.3_330)] text-[oklch(0.05_0.02_290)] hover:scale-105 transition-all duration-200 pulse-neon font-semibold"
      onClick={() => {
        if (!address) openConnectModal && openConnectModal();
      }}
    >
      <Wallet className="mr-2 h-4 w-4" />
      Connect Wallet
    </Button>
  );
}
