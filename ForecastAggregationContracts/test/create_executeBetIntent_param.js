import { ethers } from "ethers";

// ===== 配置区 =====
const privateKey = "0x0x1xxxxx"; // 进行调用合约的钱包私钥
const verifyingContract = "0xE61c16E884DaF7Fab9586a52d3468F409f0A3Ec0"; // BetRouterUpgradeable合约地址
const chainId = 11155111;
const topicId = "0x11111111111111111111111111111111111111111111111111111111111c1234";
const amount = 30000;
const nonce = 0;
const deadline = 2000000000;

// ===== 主函数 =====
async function main() {
  const wallet = new ethers.Wallet(privateKey);

  // 1️⃣ EIP712 domain
  const domain = {
    name: "PredictionMarketAggregator",
    version: "1",
    chainId,
    verifyingContract,
  };

  // 2️⃣ Solidity 对应的 struct typehash
  const BET_INTENT_TYPEHASH = ethers.keccak256(
    ethers.toUtf8Bytes(
      "BetIntent(address user,bytes32 topicId,uint256 amount,uint256 nonce,uint256 deadline)"
    )
  );

  // 3️⃣ struct hash
  // 对应 Solidity 中的 `keccak256(abi.encode(...))`
  // 先 abi.encode(BET_INTENT_TYPEHASH, ...struct fields) 再 keccak256
  const structHash = ethers.keccak256(
    ethers.AbiCoder.defaultAbiCoder().encode(
      ["bytes32", "address", "bytes32", "uint256", "uint256", "uint256"],
      [BET_INTENT_TYPEHASH, wallet.address, topicId, amount, nonce, deadline]
    )
  );

  // 4️⃣ 定义 EIP712 类型信息（用于 signTypedData）
  // 这里方便直接使用 `wallet.signTypedData()` 生成签名
  const types = {
    BetIntent: [
      { name: "user", type: "address" },
      { name: "topicId", type: "bytes32" },
      { name: "amount", type: "uint256" },
      { name: "nonce", type: "uint256" },
      { name: "deadline", type: "uint256" }
    ]
  };
    
  const message = {
    user: wallet.address,
    topicId,
    amount,
    nonce,
    deadline
  };

  // 5️⃣ domain separator
  // 对应 Solidity 中 `_domainSeparatorV4()` 的实现
  // keccak256(abi.encode(
  //   EIP712DOMAIN_TYPEHASH,
  //   keccak256(bytes(name)),
  //   keccak256(bytes(version)),
  //   chainId,
  //   verifyingContract
  // ))
  const EIP712DOMAIN_TYPEHASH = ethers.keccak256(
    ethers.toUtf8Bytes(
      "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
    )
  );
  const domainSeparator = ethers.keccak256(
    ethers.AbiCoder.defaultAbiCoder().encode(
      ["bytes32","bytes32","bytes32","uint256","address"],
      [
        EIP712DOMAIN_TYPEHASH,
        ethers.keccak256(ethers.toUtf8Bytes(domain.name)),
        ethers.keccak256(ethers.toUtf8Bytes(domain.version)),
        domain.chainId,
        domain.verifyingContract
      ]
    )
  );

  // 6️⃣ digest = keccak256("\x19\x01" || domainSeparator || structHash)
  const digest = ethers.keccak256(
    ethers.concat([
      "0x1901",
      domainSeparator,
      structHash
    ])
  );

  // 7️⃣ 签名 digest
  const signature = await wallet.signTypedData(domain, types, message);

  // 8️⃣ 生成 betId（保持原来的逻辑）
  const betId = ethers.keccak256(
    ethers.AbiCoder.defaultAbiCoder().encode(
      ["address", "bytes32", "uint256"],
      [wallet.address, topicId, nonce]
    )
  );

  // 9️⃣ 输出 JSON
  const intentJSON = {
    intent: [
      wallet.address,
      topicId,
      amount,
      nonce,
      deadline
      ],
    digest,
    signature,
    betId
  };

  console.log(JSON.stringify(intentJSON, null, 2));
}

main().catch(console.error);
