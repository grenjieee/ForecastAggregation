/**
 * Design: Cyberpunk Neon Futurism
 * - Platform comparison table with neon highlights
 * - Best price highlighted with green glow
 * - External link buttons with platform branding
 */

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { type Market } from "@/lib/mockData";
import { ExternalLink, TrendingDown, TrendingUp } from "lucide-react";

interface MarketDetailDialogProps {
  market: Market | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MarketDetailDialog({ market, open, onOpenChange }: MarketDetailDialogProps) {
  if (!market) return null;

  const bestYesPrice = Math.min(...market.platforms.map(p => p.yesPrice));
  const bestNoPrice = Math.min(...market.platforms.map(p => p.noPrice));

  const getPlatformColor = (platform: string) => {
    const colors = {
      Polymarket: 'from-[oklch(0.65_0.3_330)] to-[oklch(0.7_0.28_350)]',
      Kalshi: 'from-[oklch(0.85_0.25_140)] to-[oklch(0.8_0.2_200)]',
      Robinhood: 'from-[oklch(0.8_0.2_200)] to-[oklch(0.75_0.25_280)]'
    };
    return colors[platform as keyof typeof colors] || colors.Polymarket;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto neon-border-gradient">
        <DialogHeader>
          <DialogTitle className="text-2xl text-foreground pr-8">{market.title}</DialogTitle>
          <DialogDescription className="text-base text-muted-foreground">
            {market.description}
          </DialogDescription>
          <div className="flex gap-2 pt-2">
            <Badge className="bg-[oklch(0.8_0.2_200)] text-[oklch(0.05_0.02_290)]">
              {market.category}
            </Badge>
            <Badge variant="outline" className="border-[oklch(0.3_0.15_200/0.5)]">
              Ends: {new Date(market.endDate).toLocaleDateString()}
            </Badge>
            <Badge variant="outline" className="border-[oklch(0.3_0.15_200/0.5)]">
              Volume: {market.totalVolume}
            </Badge>
          </div>
        </DialogHeader>

        <div className="mt-6 space-y-6">
          <div>
            <h3 className="text-lg font-semibold mb-4 text-foreground">Platform Comparison</h3>
            <div className="space-y-3">
              {market.platforms.map((platform) => {
                const isYesBest = platform.yesPrice === bestYesPrice;
                const isNoBest = platform.noPrice === bestNoPrice;

                return (
                  <div
                    key={platform.platform}
                    className={`neon-border-gradient rounded-lg p-4 backdrop-blur-sm transition-all duration-200 hover:scale-[1.01] ${
                      isYesBest || isNoBest ? 'neon-glow-green bg-[oklch(0.15_0.08_285/0.7)]' : 'bg-[oklch(0.12_0.06_285/0.5)]'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <h4 className="text-lg font-bold data-font text-foreground">{platform.platform}</h4>
                        {(isYesBest || isNoBest) && (
                          <Badge className="bg-[oklch(0.85_0.25_140)] text-[oklch(0.05_0.02_290)] gap-1">
                            <TrendingUp className="h-3 w-3" />
                            Best Price
                          </Badge>
                        )}
                      </div>
                      <Button
                        size="sm"
                        className={`bg-gradient-to-r ${getPlatformColor(platform.platform)} text-white hover:scale-105 transition-all`}
                        onClick={() => window.open(platform.url, '_blank')}
                      >
                        Trade Now
                        <ExternalLink className="ml-2 h-4 w-4" />
                      </Button>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                      <div>
                        <div className="text-xs text-muted-foreground mb-1">YES Price</div>
                        <div className={`text-xl font-bold data-font ${isYesBest ? 'text-[oklch(0.85_0.25_140)] pulse-neon' : 'text-foreground'}`}>
                          {(platform.yesPrice * 100).toFixed(1)}%
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground mb-1">NO Price</div>
                        <div className={`text-xl font-bold data-font ${isNoBest ? 'text-[oklch(0.85_0.25_140)] pulse-neon' : 'text-foreground'}`}>
                          {(platform.noPrice * 100).toFixed(1)}%
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground mb-1">Volume</div>
                        <div className="text-sm font-semibold data-font text-foreground">{platform.volume}</div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground mb-1">Liquidity</div>
                        <div className="text-sm font-semibold data-font text-foreground">{platform.liquidity}</div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground mb-1">Fee</div>
                        <div className="text-sm font-semibold data-font text-foreground">{platform.fee}</div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="neon-border-gradient rounded-lg p-4 bg-[oklch(0.12_0.06_285/0.5)]">
            <h4 className="text-sm font-semibold mb-2 text-foreground flex items-center gap-2">
              <TrendingDown className="h-4 w-4 text-[oklch(0.8_0.2_200)]" />
              Price Spread Analysis
            </h4>
            <p className="text-sm text-muted-foreground">
              The best YES price is <span className="text-[oklch(0.85_0.25_140)] font-bold">{(bestYesPrice * 100).toFixed(1)}%</span> and 
              the worst is <span className="text-[oklch(0.7_0.28_350)] font-bold">{(Math.max(...market.platforms.map(p => p.yesPrice)) * 100).toFixed(1)}%</span>.
              By choosing the best platform, you can save up to{' '}
              <span className="text-[oklch(0.85_0.25_140)] font-bold">
                {((Math.max(...market.platforms.map(p => p.yesPrice)) - bestYesPrice) / Math.max(...market.platforms.map(p => p.yesPrice)) * 100).toFixed(1)}%
              </span>{' '}
              on your trade.
            </p>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
