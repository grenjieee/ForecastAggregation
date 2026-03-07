import { create } from "zustand";
import { type Market } from "@/lib/mockData";

interface DialogState {
    selectedMarket: Market | null;
    dialogOpen: boolean;
    openDialog: (market: Market) => void;
    closeDialog: () => void;
}

const marketDialogStore = create<DialogState>((set) => ({
    selectedMarket: null,
    dialogOpen: false,
    openDialog: (market) => set({ selectedMarket: market, dialogOpen: true }),
    closeDialog: () => set({ selectedMarket: null, dialogOpen: false }),
}));

export default marketDialogStore;