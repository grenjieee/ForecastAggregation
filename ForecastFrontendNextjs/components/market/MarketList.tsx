'use client';
import { MarketCard } from "@/components/market/MarketCard";
import { Virtuoso } from "react-virtuoso";
import type { Market, MarketQueryResult } from "@/lib/mockData";
import { useCallback, useEffect, useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import SearchStore from "@/stores/SearchStore";
import DialogState from "@/stores/dialogStore";
import { fetchMarkets } from "@/lib/api/markets";
import { Search } from "lucide-react";
import SelectedCategoryStore from "@/stores/SelectedCategoryStore";

// interface MarketListProps {
//     markets: Market[];
//     onViewDetails: (market: Market) => void;
//     height?: number;
//     onEndReached?: () => void;
//     isLoading?: boolean;
//     isFetchingNextPage?: boolean;
// }

export function MarketList(firstpage: MarketQueryResult) {
    const { searchValue } = SearchStore();
    const { value: selectedCategory } = SelectedCategoryStore();
    const { openDialog } = DialogState();

    const PAGE_SIZE = 20;
    const {
        data,
        isLoading,
        isError,
        fetchNextPage,
        hasNextPage,
        isFetchingNextPage,
    } = useInfiniteQuery({
        queryKey: ["items", selectedCategory, searchValue],
        queryFn: ({ pageParam = 2 }) => fetchMarkets("sports", "active", pageParam, PAGE_SIZE),
        initialPageParam: 2,
        getNextPageParam: (lastPage: any, allPages: any[]) => {
            if (!lastPage || !lastPage.items) return undefined;
            return lastPage.items.length === PAGE_SIZE ? allPages.length + 1 : undefined;
        },
        initialData: {
            pages: [firstpage],
            pageParams: [1],
        },
    });
    const allMarkets = data?.pages?.flatMap(page => page.items) || [];
    const filteredMarkets = allMarkets.filter((market) => {
        console.log("Filtering market:", market.type);
        const matchesCategory =
            selectedCategory === "All" || market.type === selectedCategory;
        const matchesSearch =
            market.title.toLowerCase().includes(searchValue.toLowerCase()) ||
            market.description.toLowerCase().includes(searchValue.toLowerCase());
        return matchesCategory && matchesSearch;
    });
    const totalMarkets = data?.pages?.[0]?.total || null;

    const handleViewDetails = useCallback((market: Market) => {
        openDialog(market);
    }, []);

    const [columns, setColumns] = useState(3);
    useEffect(() => {
        function handleResize() {
            if (window.innerWidth < 1024) {
                setColumns(2);
            } else {
                setColumns(3);
            }
        }
        handleResize();
        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);

    const rowCount = Math.ceil(filteredMarkets.length / columns);
    return (
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
                            {totalMarkets !== null ? `${totalMarkets} markets available` : null}
                        </p>
                    </div>
                </div>
                {filteredMarkets?.length > 0 ? (
                    <Virtuoso
                        style={{ height: '600px' }}
                        totalCount={rowCount}
                        itemContent={rowIndex => {
                            const items = [];
                            for (let col = 0; col < columns; col++) {
                                const marketIndex = rowIndex * columns + col;
                                if (marketIndex < filteredMarkets.length) {
                                    items.push(
                                        <div key={filteredMarkets[marketIndex].canonical_id} style={{ flex: 1, minWidth: 0, marginRight: 12, marginLeft: 12 }}>
                                            <MarketCard
                                                market={filteredMarkets[marketIndex]}
                                                onViewDetails={handleViewDetails}
                                            />
                                        </div>
                                    );
                                } else {
                                    items.push(<div key={"empty-" + col} style={{ flex: 1, minWidth: 0 }} />);
                                }
                            }
                            return (
                                <div style={{ display: 'flex', gap: 0, marginTop: 10, marginBottom: 10 }}>
                                    {items}
                                </div>
                            );
                        }}
                        endReached={() => {
                            if (hasNextPage && !isFetchingNextPage) fetchNextPage();
                        }}
                        components={{
                            Footer: () => (
                                isFetchingNextPage ? <div style={{ textAlign: 'center', padding: 16 }}>加载中...</div> : null
                            ),
                        }}
                    />
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
    );
}


