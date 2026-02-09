"use client";
/**
 * Design: Cyberpunk Neon Futurism
 * - Hero section with animated neon grid background
 * - Category filters with neon highlights
 * - Market cards in responsive grid
 * - Sticky header with glassmorphism effect
 */

import { MarketCard } from "@/components/MarketCard";
import { MarketDetailDialog } from "@/components/MarketDetailDialog";
import { WalletConnect } from "@/components/WalletConnect";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { categories, mockMarkets, type Market } from "@/lib/mockData";
import { Search, TrendingUp } from "lucide-react";
import { useState } from "react";
import { useBearStore, bearState } from "@/stores/BearStore";

export default function Home() {
  const [selectedCategory, setSelectedCategory] = useState<string>("All");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedMarket, setSelectedMarket] = useState<Market | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const filteredMarkets = mockMarkets.filter((market) => {
    const matchesCategory =
      selectedCategory === "All" || market.category === selectedCategory;
    const matchesSearch =
      market.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      market.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  const handleViewDetails = (market: Market) => {
    setSelectedMarket(market);
    setDialogOpen(true);
  };
  // const bears = useBearStore((state) => {
  //   console.log(state);
  //   return state.bears
  // })
  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="sticky top-0 z-50 backdrop-blur-md bg-[oklch(0.1_0.06_285/0.8)] border-b border-[oklch(0.3_0.15_200/0.3)]">
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
      </header>
      {/* {bears} */}
      {/* Hero Section */}
      <section
        className="relative py-20 overflow-hidden"
        style={{
          backgroundImage: `url('https://private-us-east-1.manuscdn.com/sessionFile/Y4jxdV6ZTyB2U4L7qNAagR/sandbox/Ob3s0izseLi8hbiYELWCH6-img-1_1770390717000_na1fn_aGVyby1iYWNrZ3JvdW5k.png?x-oss-process=image/resize,w_1920,h_1920/format,webp/quality,q_80&Expires=1798761600&Policy=eyJTdGF0ZW1lbnQiOlt7IlJlc291cmNlIjoiaHR0cHM6Ly9wcml2YXRlLXVzLWVhc3QtMS5tYW51c2Nkbi5jb20vc2Vzc2lvbkZpbGUvWTRqeGRWNlpUeUIyVTRMN3FOQWFnUi9zYW5kYm94L09iM3MwaXpzZUxpOGhiaVlFTFdDSDYtaW1nLTFfMTc3MDM5MDcxNzAwMF9uYTFmbl9hR1Z5YnkxaVlXTnJaM0p2ZFc1ay5wbmc~eC1vc3MtcHJvY2Vzcz1pbWFnZS9yZXNpemUsd18xOTIwLGhfMTkyMC9mb3JtYXQsd2VicC9xdWFsaXR5LHFfODAiLCJDb25kaXRpb24iOnsiRGF0ZUxlc3NUaGFuIjp7IkFXUzpFcG9jaFRpbWUiOjE3OTg3NjE2MDB9fX1dfQ__&Key-Pair-Id=K2HSFNDJXOU9YS&Signature=IIawJTZ2tubA8rY9UdMMhSmI74mtY6t-wpDnmyckItpJgn8GqLi9d3QZ1-QtGQgSzRbh0h~DwA5ZEtPldnySZ~b~5XxHsN~5C2uETQO9sSVDVRPfnR4Y2xXQrCO~C~HzH0xNFC8tLkIjw~UzEp~ziC04qTiabxcaWKjaDCwjohw8P2Rs8Nr7Cb-YP6TROm~6W3TkkYf1PA6XM2Fq-fWL323a98BIuFoS4Nm41Df84JQFdobqVCmnP49yUyi8RSe7X-jwmc5tRTdVmfcCWm3OYKePaUvg3tFraq9quvVaN5PdlNDsOTuQVCZ71HK1gynm0Ph7iFnIEHJGy83LbkwtQA__')`,
          backgroundSize: "cover",
          backgroundPosition: "center",
        }}
      >
        <div className="absolute inset-0 bg-gradient-to-b from-[oklch(0.08_0.05_290/0.7)] to-[oklch(0.08_0.05_290)]" />
        <div className="container relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <h2 className="text-5xl md:text-6xl font-bold text-foreground leading-tight">
              Find the Best Prices Across{" "}
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-[oklch(0.8_0.2_200)] via-[oklch(0.65_0.3_330)] to-[oklch(0.7_0.28_350)]">
                All Markets
              </span>
            </h2>
            <p className="text-xl text-muted-foreground">
              Compare prediction markets from Polymarket, Kalshi, Robinhood and
              more. Save money by finding the best odds for every event.
            </p>

            {/* Search Bar */}
            <div className="relative max-w-2xl mx-auto mt-8">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
              <Input
                placeholder="Search markets..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-12 h-14 text-lg neon-border-gradient bg-[oklch(0.12_0.06_285/0.8)] backdrop-blur-sm border-0 focus-visible:ring-[oklch(0.8_0.2_200)]"
              />
            </div>
          </div>
        </div>
      </section>

      {/* Category Filters */}
      <section className="border-b border-[oklch(0.3_0.15_200/0.3)] bg-[oklch(0.1_0.06_285/0.5)] backdrop-blur-sm sticky top-[73px] z-40">
        <div className="container py-4">
          <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-hide">
            {categories.map((category) => (
              <Button
                key={category}
                variant={selectedCategory === category ? "default" : "outline"}
                size="sm"
                onClick={() => setSelectedCategory(category)}
                className={
                  selectedCategory === category
                    ? "bg-gradient-to-r from-[oklch(0.8_0.2_200)] to-[oklch(0.65_0.3_330)] text-[oklch(0.05_0.02_290)] hover:scale-105 transition-all neon-glow-cyan"
                    : "border-[oklch(0.3_0.15_200/0.5)] hover:border-[oklch(0.8_0.2_200)] hover:text-[oklch(0.8_0.2_200)] transition-all"
                }
              >
                {category}
              </Button>
            ))}
          </div>
        </div>
      </section>

      {/* Markets Grid */}
      <section className="py-12">
        <div className="container">
          <div className="flex items-center justify-between mb-8">
            <div>
              <h3 className="text-2xl font-bold text-foreground mb-2">
                {selectedCategory === "All"
                  ? "All Markets"
                  : `${selectedCategory} Markets`}
              </h3>
              <p className="text-muted-foreground">
                {filteredMarkets.length}{" "}
                {filteredMarkets.length === 1 ? "market" : "markets"} available
              </p>
            </div>
          </div>

          {filteredMarkets.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {filteredMarkets.map((market) => (
                <MarketCard
                  key={market.id}
                  market={market}
                  onViewDetails={handleViewDetails}
                />
              ))}
            </div>
          ) : (
            <div className="text-center py-16">
              <div className="neon-border-gradient rounded-lg p-12 max-w-md mx-auto bg-[oklch(0.12_0.06_285/0.5)]">
                <Search className="h-16 w-16 text-muted-foreground mx-auto mb-4" />
                <h4 className="text-xl font-semibold text-foreground mb-2">
                  No markets found
                </h4>
                <p className="text-muted-foreground">
                  Try adjusting your search or category filter
                </p>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-[oklch(0.3_0.15_200/0.3)] bg-[oklch(0.1_0.06_285/0.5)] backdrop-blur-sm py-8 mt-12">
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
      </footer>

      {/* Market Detail Dialog */}
      <MarketDetailDialog
        market={selectedMarket}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </div>
  );
}
