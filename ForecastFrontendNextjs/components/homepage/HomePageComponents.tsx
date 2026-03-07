import { Search, TrendingUp } from "lucide-react";
import { WalletConnect } from "../WalletConnect"; 

export function Header() {
    return <header className="sticky top-0 z-50 backdrop-blur-md bg-[oklch(0.1_0.06_285/0.8)] border-b border-[oklch(0.3_0.15_200/0.3)]">
        <div className="container py-4">
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <div className="relative">
                        <TrendingUp className="h-8 w-8 text-[oklch(0.8_0.2_200)]" />
                        <div className="absolute inset-0 blur-lg bg-[oklch(0.8_0.2_200/0.5)]" />
                    </div>
                    <h1 className="text-2xl font-bold text-foreground">
                        Prediction
                        <span className="text-[oklch(0.8_0.2_200)]">Market</span>
                    </h1>
                </div>
                <WalletConnect />
            </div>
        </div>
    </header>;
}


export function Footer() {
    return <footer className="border-t border-[oklch(0.3_0.15_200/0.3)] bg-[oklch(0.1_0.06_285/0.5)] backdrop-blur-sm py-8 mt-12">
        <div className="container">
            <div className="text-center text-sm text-muted-foreground">
                <p className="mb-2">
                    Prediction Market Aggregator - Compare prices across multiple
                    platforms
                </p>
                <p className="text-xs">
                    This is a demo application. Market data is simulated for
                    demonstration purposes.
                </p>
            </div>
        </div>
    </footer>;
}