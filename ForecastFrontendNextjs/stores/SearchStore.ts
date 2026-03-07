import { create } from "zustand";

interface SearchState {
    searchValue: string;
    setSearchValue: (value: string) => void;
}

const SearchStore = create<SearchState>((set) => ({
    searchValue: "",
    setSearchValue: (value) => set({ searchValue: value }),
}));

export default SearchStore;