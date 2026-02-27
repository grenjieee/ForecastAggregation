
import { MarketCard } from "@/components/MarketCard";
import { Virtuoso } from "react-virtuoso";
import type { Market } from "@/lib/mockData";
import { useEffect, useState } from "react";

interface MarketListProps {
    markets: Market[];
    onViewDetails: (market: Market) => void;
    height?: number;
    onEndReached?: () => void;
    isLoading?: boolean;
    isFetchingNextPage?: boolean;
}

export function MarketList({ markets, onViewDetails, height = 600, onEndReached, isLoading, isFetchingNextPage }: MarketListProps) {
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

    const rowCount = Math.ceil(markets.length / columns);
    return (
        <Virtuoso
            style={{ height }}
            totalCount={rowCount}
            itemContent={rowIndex => {
                const items = [];
                for (let col = 0; col < columns; col++) {
                    const marketIndex = rowIndex * columns + col;
                    if (marketIndex < markets.length) {
                        items.push(
                            <div key={markets[marketIndex].canonical_id} style={{ flex: 1, minWidth: 0 }}>
                                <MarketCard
                                    market={markets[marketIndex]}
                                    onViewDetails={onViewDetails}
                                />
                            </div>
                        );
                    } else {
                        items.push(<div key={"empty-" + col} style={{ flex: 1, minWidth: 0 }} />);
                    }
                }
                return (
                    <div style={{ display: 'flex', gap: 12, marginTop: 12, marginBottom: 12 }}>
                        {items}
                    </div>
                );
            }}
            endReached={onEndReached}
            components={{
                Footer: () => (
                    isFetchingNextPage ? <div style={{ textAlign: 'center', padding: 16 }}>加载中...</div> : null
                ),
            }}
        />
    );
}
