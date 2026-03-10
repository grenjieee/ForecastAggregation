"use client";

import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@radix-ui/react-dialog";
import { DialogHeader } from "../ui/dialog";
import { useEffect, useState } from "react";
import { Input } from "../ui/input";
import { useTokenBalance } from "@/lib/web3api";
import { fetchMarkets, fetchOrders, FetchOrdersRespone } from "@/lib/api/markets";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Virtuoso } from "react-virtuoso";

export default function BetHistoryModal({ address, onClose }: { address: string; onClose: () => void }) {

    const fetchOrdersWithPagination = async ({ pageParam = 1 }: { pageParam?: number }) => {
        const params = { wallet: address, page: pageParam };
        const response: FetchOrdersRespone = await fetchOrders(params);
        return response;
    };

    const [hasItems, setHasItems] = useState(false);

    const {
        data,
        fetchNextPage,
        hasNextPage,
        isFetchingNextPage,
        isSuccess,
    } = useInfiniteQuery({
        queryKey: ["fetchOrders", address],
        queryFn: ({ pageParam = 1 }) => fetchOrders({ wallet: address, page: pageParam }),
        getNextPageParam: (lastPage) => {
            const nextPage = lastPage.page + 1;
            return nextPage <= Math.ceil(lastPage.total / lastPage.pageSize) ? nextPage : undefined;
        },
        initialPageParam: 1, // 添加初始页码
    });

    useEffect(() => {
        if (isSuccess && data?.pages?.[0]?.items?.length > 0) {
            setHasItems(true);
        } else {
            setHasItems(false);
        }
    }, [data, isSuccess]);

    useEffect(() => {
        // Disable scrolling when the modal is open
        document.body.style.overflow = "hidden";
        return () => {
            // Re-enable scrolling when the modal is closed
            document.body.style.overflow = "";
        };
    }, []);
    return (
        <Dialog open={true} onOpenChange={onClose}>
            <DialogContent className="fixed flex items-center justify-center inset-0 z-50 bg-[oklch(0.1_0.05_250/0.9)] backdrop-blur-md pointer-events-auto">
                <div className="w-300 flex flex-col bg-[oklch(0.12_0.06_285/0.5)] neon-border-gradient rounded-lg shadow-lg p-8 overflow-y-auto text-center">
                    <DialogHeader className="w-full">
                        <DialogTitle className="text-3xl font-bold text-foreground border-b border-[oklch(0.3_0.15_200/0.5)] pb-2 flex justify-between items-center">
                            订单列表
                            <button onClick={onClose} className="text-lg font-bold text-red-500 hover:text-red-700 transition-all">
                                X
                            </button>
                        </DialogTitle>
                        <DialogDescription className="text-base text-muted-foreground mt-2">
                            {address}
                        </DialogDescription>
                    </DialogHeader>

                    {!hasItems && <p className="text-sm text-muted-foreground mt-4">没有订单数据。</p>}
                    <Virtuoso
                        style={{ height: "400px" }}
                        data={data?.pages.flatMap((page) => (page as FetchOrdersRespone).items) || []}
                        endReached={() => {
                            if (hasNextPage && !isFetchingNextPage) {
                                fetchNextPage();
                            }
                        }}
                        itemContent={(index, order) => (
                            <div key={order.orderUuid} className="p-2 border-b border-gray-200">
                                <p>订单号: {order.orderUuid}</p>
                                <p>金额: {order.betAmount}</p>
                                <p>状态: {order.status}</p>
                            </div>
                        )}
                    />

                    {isFetchingNextPage && <p>加载更多...</p>}
                </div>
            </DialogContent>
        </Dialog>
    );
}
