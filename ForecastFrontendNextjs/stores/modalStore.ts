import { create } from "zustand";

const MODAL_IDS = {
    BET_HISTORY: "betHistory",
    PLAYER_RECHARGE: "playerRecharge",
    // 其他弹窗 ID 可以在这里添加
};

interface Modal {
    id: string;
    props?: Record<string, any>;
}

interface ModalState {
    modal: Modal | null;
    openModal: (id: string, props?: Record<string, any>) => void;
    openBetHistoryModal: (address: string) => void;
    openPlayerRechargeModal: (address: string) => void;
    closeModal: () => void;
}

export const useModalStore = create<ModalState>((set) => ({
    modal: null,
    openModal: (id, props) =>
        set(() => ({
            modal: { id, props },
        })),
    openBetHistoryModal: (address) =>
        set(() => ({
            modal: { id: MODAL_IDS.BET_HISTORY, props: { address } },
        })),
    openPlayerRechargeModal: (address) =>
        set(() => ({
            modal: { id: MODAL_IDS.PLAYER_RECHARGE, props: { address } },
        })),
    closeModal: () =>
        set(() => ({
            modal: null,
        })),
}));

export { MODAL_IDS };