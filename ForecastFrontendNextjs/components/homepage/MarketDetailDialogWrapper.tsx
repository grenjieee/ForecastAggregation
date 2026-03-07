"use client";

import useDialogStore from "@/stores/dialogStore";
import { MarketDetailDialog } from "@/components/MarketDetailDialog";

export default function MarketDetailDialogWrapper() {
    const { selectedMarket, dialogOpen, closeDialog, openDialog } = useDialogStore();

    return (
        <MarketDetailDialog
            market={selectedMarket}
            open={dialogOpen}
            onOpenChange={(isOpen) => {
                if (!isOpen) closeDialog();
            }}
        />
    );
}