'use client';

import { Search } from "lucide-react";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { categories } from "@/lib/mockData";
import SearchStore from "@/stores/SearchStore";
import SelectedCategoryStore from "@/stores/SelectedCategoryStore";

export function Sectionsraech() {
    const { searchValue, setSearchValue } = SearchStore();

    return <div className="relative max-w-2xl mx-auto mt-8">
        <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
        <Input
            placeholder="Search markets..."
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            className="pl-12 h-14 text-lg neon-border-gradient bg-[oklch(0.12_0.06_285/0.8)] backdrop-blur-sm border-0 focus-visible:ring-[oklch(0.8_0.2_200)]"
        />
    </div>
}

export function Category() {
    const { value, setValue } = SelectedCategoryStore();
    return <section className="border-b border-[oklch(0.3_0.15_200/0.3)] bg-[oklch(0.1_0.06_285/0.5)] backdrop-blur-sm sticky top-[73px] z-40">
        <div className="container py-4">
            <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-hide">
                {categories.map((category) => (
                    <Button
                        key={category}
                        variant={value === category ? "default" : "outline"}
                        size="sm"
                        onClick={() => setValue(category)}
                        className={
                            value === category
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
}