import { ethers } from "ethers";

// 1️⃣ 私钥账号
const PRIVATE_KEY = "0x1xxxxx"; // 注意：进行签名的前端钱包私钥
const wallet = new ethers.Wallet(PRIVATE_KEY);
console.log("wallet:", wallet.address);

// 2️⃣ 准备参数
const betId = "0x7bfe0c9f6f8c8434363ce46d4fb6e969183889b45b101c2f371b6a21e85863bb"; // 示例 betId
const status = 2; // 对应 BetStatus 枚举，例如 FUNDS_LOCKED = 2
const signerNonce = 1; // 你的 nonce，根据链上记录填

// 3️⃣ 生成 digest（与链上相同算法）
const digest = ethers.keccak256(
    ethers.solidityPacked(
        ["bytes32", "uint8", "uint256"],
        [betId, status, signerNonce]
    )
);

console.log("Digest:", digest);


// 4️⃣ 直接签 digest（⚠️不是 signMessage）
const sig = wallet.signingKey.sign(digest);

// 5️⃣ 转换成标准 signature
const signature = ethers.Signature.from(sig).serialized;

console.log("Signature:", signature);
