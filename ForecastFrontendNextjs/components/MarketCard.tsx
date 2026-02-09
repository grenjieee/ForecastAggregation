/**
 * Design: Cyberpunk Neon Futurism
 * - Neon gradient border with glow effect
 * - Semi-transparent card background
 * - Pulse animation for probability numbers
 * - Hover effect: enhanced glow and slight lift
 */

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getBestPrice, type Market } from "@/lib/mockData";
import { ArrowRight, TrendingUp } from "lucide-react";
import { useState } from "react";

interface MarketCardProps {
  market: Market;
  onViewDetails: (market: Market) => void;
}

export function MarketCard({ market, onViewDetails }: MarketCardProps) {
  const [isHovered, setIsHovered] = useState(false);
  const bestPrice = getBestPrice(market);
  const bestPlatform = market.platforms.find(
    p => p.platform === bestPrice.platform
  );

  const getCategoryColor = (category: string) => {
    const colors = {
      Politics: "bg-[oklch(0.65_0.3_330)] text-[oklch(0.95_0.02_290)]",
      Sports: "bg-[oklch(0.85_0.25_140)] text-[oklch(0.05_0.02_290)]",
      Crypto: "bg-[oklch(0.8_0.2_200)] text-[oklch(0.05_0.02_290)]",
      Finance: "bg-[oklch(0.7_0.28_350)] text-[oklch(0.95_0.02_290)]",
      Tech: "bg-[oklch(0.7_0.2_60)] text-[oklch(0.05_0.02_290)]",
      Culture: "bg-[oklch(0.75_0.25_280)] text-[oklch(0.95_0.02_290)]",
    };
    return colors[category as keyof typeof colors] || colors.Crypto;
  };

  return (
    <Card
      className={`neon-border-gradient backdrop-blur-sm transition-all duration-300 cursor-pointer ${
        isHovered ? "neon-glow-cyan scale-[1.02] -translate-y-1" : ""
      }`}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      onClick={() => onViewDetails(market)}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-2">
          <div className="flex-1">
            <CardTitle className="text-lg mb-2 text-foreground leading-tight">
              {market.title}
            </CardTitle>
            <CardDescription className="text-sm text-muted-foreground line-clamp-2">
              {market.description}
            </CardDescription>
          </div>
          <Badge className={`${getCategoryColor(market.category)} shrink-0`}>
            {market.category}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Best Price Highlight */}
        <div className="neon-border-gradient rounded-lg p-4 bg-[oklch(0.15_0.08_285/0.5)] backdrop-blur-sm">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-muted-foreground uppercase tracking-wider">
              Best Price
            </span>
            <div className="flex items-center gap-1 text-[oklch(0.85_0.25_140)]">
              <TrendingUp className="h-3 w-3" />
              <span className="text-xs font-semibold">
                Save {bestPrice.savings}
              </span>
            </div>
          </div>

          <div className="flex items-end justify-between">
            <div>
              <div className="text-xs text-muted-foreground mb-1">
                {bestPlatform?.platform}
              </div>
              <div className="flex items-baseline gap-3">
                <div>
                  <span className="text-xs text-muted-foreground mr-1">
                    YES
                  </span>
                  <span className="text-2xl font-bold data-font text-[oklch(0.85_0.25_140)] pulse-neon">
                    {(bestPrice.yesPrice * 100).toFixed(0)}%
                  </span>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground mr-1">NO</span>
                  <span className="text-2xl font-bold data-font text-[oklch(0.7_0.28_350)]">
                    {((1 - bestPrice.yesPrice) * 100).toFixed(0)}%
                  </span>
                </div>
              </div>
            </div>

            <Button
              size="sm"
              variant="ghost"
              className="text-[oklch(0.8_0.2_200)] hover:text-[oklch(0.85_0.22_200)] hover:bg-[oklch(0.8_0.2_200/0.1)] md:text-[0.6rem]"
              onClick={e => {
                e.stopPropagation();
                onViewDetails(market);
              }}
            >
              Compare
              <ArrowRight className="ml-1 h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Market Stats */}
        <div className="grid grid-cols-3 gap-3 text-center">
          <div>
            <div className="text-xs text-muted-foreground mb-1">Volume</div>
            <div className="text-sm font-semibold data-font text-foreground">
              {market.totalVolume}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted-foreground mb-1">Platforms</div>
            <div className="text-sm font-semibold data-font text-foreground">
              {market.platforms.length}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted-foreground mb-1">Ends</div>
            <div className="text-sm font-semibold data-font text-foreground">
              {new Date(market.endDate).toLocaleDateString("en-US", {
                month: "short",
                day: "numeric",
              })}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
