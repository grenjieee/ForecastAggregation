import { useContractRead } from "wagmi";
import { formatUnits } from "ethers";
import { createPublicClient, http } from "viem";
import { sepolia } from "./chains";
const client = createPublicClient({
    chain: sepolia,
    transport: http()
})

//代币余额查询
export const TOKEN_ADDRESS = "0x85166220421C86B90a630E496840d6C38aa7455B"; //代币地址
export const TOKEN_ABI = [
    {
        type: 'event',
        name: 'Approval',
        inputs: [
            {
                indexed: true,
                name: 'owner',
                type: 'address',
            },
            {
                indexed: true,
                name: 'spender',
                type: 'address',
            },
            {
                indexed: false,
                name: 'value',
                type: 'uint256',
            },
        ],
    },
    {
        type: 'event',
        name: 'Transfer',
        inputs: [
            {
                indexed: true,
                name: 'from',
                type: 'address',
            },
            {
                indexed: true,
                name: 'to',
                type: 'address',
            },
            {
                indexed: false,
                name: 'value',
                type: 'uint256',
            },
        ],
    },
    {
        type: 'function',
        name: 'allowance',
        stateMutability: 'view',
        inputs: [
            {
                name: 'owner',
                type: 'address',
            },
            {
                name: 'spender',
                type: 'address',
            },
        ],
        outputs: [
            {
                type: 'uint256',
            },
        ],
    },
    {
        type: 'function',
        name: 'approve',
        stateMutability: 'nonpayable',
        inputs: [
            {
                name: 'spender',
                type: 'address',
            },
            {
                name: 'amount',
                type: 'uint256',
            },
        ],
        outputs: [
            {
                type: 'bool',
            },
        ],
    },
    {
        type: 'function',
        name: 'balanceOf',
        stateMutability: 'view',
        inputs: [
            {
                name: 'account',
                type: 'address',
            },
        ],
        outputs: [
            {
                type: 'uint256',
            },
        ],
    },
    {
        type: 'function',
        name: 'decimals',
        stateMutability: 'view',
        inputs: [],
        outputs: [
            {
                type: 'uint8',
            },
        ],
    },
    {
        type: 'function',
        name: 'name',
        stateMutability: 'view',
        inputs: [],
        outputs: [
            {
                type: 'string',
            },
        ],
    },
    {
        type: 'function',
        name: 'symbol',
        stateMutability: 'view',
        inputs: [],
        outputs: [
            {
                type: 'string',
            },
        ],
    },
    {
        type: 'function',
        name: 'totalSupply',
        stateMutability: 'view',
        inputs: [],
        outputs: [
            {
                type: 'uint256',
            },
        ],
    },
    {
        type: 'function',
        name: 'transfer',
        stateMutability: 'nonpayable',
        inputs: [
            {
                name: 'recipient',
                type: 'address',
            },
            {
                name: 'amount',
                type: 'uint256',
            },
        ],
        outputs: [
            {
                type: 'bool',
            },
        ],
    },
    {
        type: 'function',
        name: 'transferFrom',
        stateMutability: 'nonpayable',
        inputs: [
            {
                name: 'sender',
                type: 'address',
            },
            {
                name: 'recipient',
                type: 'address',
            },
            {
                name: 'amount',
                type: 'uint256',
            },
        ],
        outputs: [
            {
                type: 'bool',
            },
        ],
    },
] as const
export async function useTokenBalance(userAddress: string) {
    try {
        // 同时获取余额和小数位数
        const [balance, decimals] = await Promise.all([
            client.readContract({
                address: TOKEN_ADDRESS as `0x${string}`,
                abi: TOKEN_ABI,
                functionName: 'balanceOf',
                args: [userAddress as `0x${string}`]
            }) as Promise<bigint>, // 明确指定返回值类型为 bigint
            client.readContract({
                address: TOKEN_ADDRESS as `0x${string}`,
                abi: TOKEN_ABI,
                functionName: 'decimals'
            }) as Promise<number>, // 明确指定返回值类型为 number
        ]);

        // 将原始余额转换为正确的代币金额
        const formattedBalance = formatUnits(balance, decimals);


        return {
            raw: balance,           // 原始余额 (BigInt)
            formatted: formattedBalance, // 格式化余额 (字符串)
            decimals: decimals,     // 小数位数
        };
    } catch (error) {
        console.error('获取代币余额失败:', error);
        return null;
    }

}


