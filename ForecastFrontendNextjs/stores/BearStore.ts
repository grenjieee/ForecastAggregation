import { create } from "zustand";
export interface bearState {
  bears: number;
  increasePopulation: () => void;
  removeAllBears: () => void;
}
export const useBearStore = create<bearState>((set) => ({
  bears: 0,
  increasePopulation: () => set((state: bearState) => ({ bears: state.bears + 1 })),
  removeAllBears: () => set({ bears: 0 }),
}));
