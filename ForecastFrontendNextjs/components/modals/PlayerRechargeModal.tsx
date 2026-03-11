"use client";

import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@radix-ui/react-dialog";
import { DialogHeader } from "../ui/dialog";
import { useEffect, useState } from "react";
import { Input } from "../ui/input";
import { useTokenBalance } from "@/lib/web3api";

export default function PlayerRechargeModal({ address, onClose }: { address: string; onClose: () => void }) {
    const [amount, setAmount] = useState("");
    const { data: balance, error } = useTokenBalance(address);

    useEffect(() => {
        if (error) {
            console.error("Error fetching token balance:", error);
        }
    }, [error]);

    useEffect(() => {
        // Disable scrolling when the modal is open
        document.body.style.overflow = "hidden";
        return () => {
            // Re-enable scrolling when the modal is closed
            document.body.style.overflow = "";
        };
    }, []);
    if (!balance) return null;
    return (
        <Dialog open={true} onOpenChange={onClose}>
            <DialogContent className="fixed flex items-center justify-center inset-0 z-50 bg-[oklch(0.1_0.05_250/0.9)] backdrop-blur-md pointer-events-auto">
                <div className="w-150  flex flex-col   bg-[oklch(0.12_0.06_285/0.5)] neon-border-gradient rounded-lg shadow-lg p-8 overflow-y-auto text-center">
                    <DialogHeader className="w-full">
                        <DialogTitle className="text-3xl font-bold text-foreground border-b border-[oklch(0.3_0.15_200/0.5)] pb-2 flex justify-between items-center">
                            充值界面
                            <button onClick={onClose} className="text-lg font-bold text-red-500 hover:text-red-700 transition-all">
                                X
                            </button>
                        </DialogTitle>
                        <DialogDescription className="text-base text-muted-foreground mt-2">
                            <div>
                                余额：{balance}
                            </div>
                        </DialogDescription>
                        <div className="relative max-w-2xl mx-auto mt-8">
                            <Input
                                placeholder="请输入金额..."
                                value={amount}
                                onChange={(e) => setAmount(e.target.value)}
                                className="h-14 text-lg neon-border-gradient bg-[oklch(0.12_0.06_285/0.8)] backdrop-blur-sm border-0 focus-visible:ring-[oklch(0.8_0.2_200)]"
                            />
                        </div>
                    </DialogHeader>

                    <div className="mt-6 space-y-4">
                        <p className="text-sm text-muted-foreground">
                            请确认充值信息后点击下方按钮完成充值。
                        </p>
                        <button
                            onClick={onClose}
                            className="w-full bg-red-500 text-white px-4 py-2 rounded-lg hover:bg-red-600 transition-all"
                        >
                            关闭
                        </button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}