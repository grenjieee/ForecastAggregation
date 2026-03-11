import { useContractRead } from "wagmi";
import { ethers } from "ethers";
import { createPublicClient, createWalletClient, http } from "viem";
import { sepolia } from "./chains";
import { useQuery } from "@tanstack/react-query";
import { TOKEN_ABI, ESCROW_VAULT_ABI } from './abis';
import { useAccount } from 'wagmi';

const client = createPublicClient({
    chain: sepolia,
    transport: http()
})

//代币余额查询
const TOKEN_ADDRESS = "0x6fa98CCFC8E55c9Bf88cf0E2Be0E7d9842dA29DB";
export const useTokenBalance = (userAddress: string) => {
    return useQuery({
        queryKey: ['tokenBalance', TOKEN_ADDRESS, userAddress],
        queryFn: async () => {
            if (!TOKEN_ADDRESS || !userAddress) {
                throw new Error('Token address and user address are required');
            }

            // 使用 viem 创建合约实例
            const balance = await client.readContract({
                address: TOKEN_ADDRESS,
                abi: TOKEN_ABI,
                functionName: 'balanceOf',
                args: [userAddress],
            }) as bigint; // 添加类型断言

            // 获取代币小数位数
            const decimals = await client.readContract({
                address: TOKEN_ADDRESS,
                abi: TOKEN_ABI,
                functionName: 'decimals',
            }) as number; // 添加类型断言

            // 格式化余额为可读数字
            return Number(balance) / Math.pow(10, decimals);
        }
    });
};
const EscrowVaultUpgradeable_ADDRESS = "0x4d164Ba20F1390aC0EDDA79FcC0eE7c165394F97";


// 锁定资金
export const lockFunds = async (userAddress: string, betId: string, amount: bigint, signature: string) => {
    if (!EscrowVaultUpgradeable_ADDRESS) {
        throw new Error('EscrowVaultUpgradeable address is required');
    }
    try {
        const walletClient = createWalletClient({
            chain: sepolia,
            transport: http(),
            account: userAddress as `0x${string}`, // 将传入的 userAddress 转换为 address 类型
        });
        const result = await walletClient.writeContract({
            address: EscrowVaultUpgradeable_ADDRESS,
            abi: ESCROW_VAULT_ABI,
            functionName: 'lockFunds',
            args: [betId, amount, signature],
        });

        return result;
    } catch (error) {
        console.error('Error locking funds:', error);
        throw error;
    }
};
