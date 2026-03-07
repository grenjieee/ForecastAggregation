import { create } from "zustand";

interface SelectedCategoryState {
    value: string;
    setValue: (value: string) => void;
}

const SelectedCategoryStore = create<SelectedCategoryState>((set) => ({
    value: "All",
    setValue: (value) => set({ value: value }),
}));

export default SelectedCategoryStore;