"use client";

import { lazy, Suspense } from "react";
import { useModalStore } from "@/stores/modalStore";

// 动态导入新创建的组件
const LazyBetHistoryModal = lazy(() => import("@/components/modals/BetHistoryModal"));
const LazyPlayerRechargeModal = lazy(() => import("@/components/modals/PlayerRechargeModal"));

export default function ModalManager() {
    const { modal, closeModal } = useModalStore();

    if (!modal) return null; // 如果没有 modal，直接返回 null


    let ModalComponent: React.ComponentType<any>; // 使用 any 允许动态类型
    let modalProps: any = { ...modal.props, onClose: closeModal }; // 合并 props 和 onClose
    // 根据 modal.id 动态选择组件
    switch (modal.id) {
        case "betHistory":
            ModalComponent = LazyBetHistoryModal;
            break;
        case "playerRecharge":
            ModalComponent = LazyPlayerRechargeModal;
            break;
        default:
            return null; // 如果 modal.id 不匹配，返回 null
    }

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ModalComponent {...modalProps} />
        </Suspense>
    );
}

