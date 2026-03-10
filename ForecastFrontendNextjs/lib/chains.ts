import { Assign, Chain, ChainFormatters, Prettify } from 'viem'

export function defineChain<
    formatters extends ChainFormatters,
    const chain extends Chain<formatters>,
>(chain: chain): Prettify<Assign<Chain<undefined>, chain>> {
    return {
        formatters: undefined,
        fees: undefined,
        serializers: undefined,
        ...chain,
    } as Assign<Chain<undefined>, chain>
}


export const sepolia = /*#__PURE__*/ defineChain({
    id: 11_155_111,
    name: 'Sepolia',
    nativeCurrency: { name: 'Sepolia Ether', symbol: 'ETH', decimals: 18 },
    rpcUrls: {
        default: {
            http: ['https://11155111.rpc.thirdweb.com'],
        },
    },
    blockExplorers: {
        default: {
            name: 'Etherscan',
            url: 'https://sepolia.etherscan.io',
            apiUrl: 'https://api-sepolia.etherscan.io/api',
        },
    },
    contracts: {
        multicall3: {
            address: '0xca11bde05977b3631167028862be2a173976ca11',
            blockCreated: 751532,
        },
        ensUniversalResolver: {
            address: '0xeeeeeeee14d718c2b47d9923deab1335e144eeee',
            blockCreated: 8_928_790,
        },
    },
    testnet: true,
})

export const ethereum = /*#__PURE__*/ defineChain({
    id: 1,
    name: 'Ethereum',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
    rpcUrls: {
        default: {
            http: ['https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID'],
        },
    },
    blockExplorers: {
        default: {
            name: 'Etherscan',
            url: 'https://etherscan.io',
            apiUrl: 'https://api.etherscan.io/api',
        },
    },
    contracts: {
        multicall3: {
            address: '0xca11bde05977b3631167028862be2a173976ca11',
            blockCreated: 12345678, // Replace with the actual block number
        },
    },
    testnet: false,
})
