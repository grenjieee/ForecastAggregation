// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

interface IBetRouter {

    /* ========== Bet Intent ========== */

    /**
     * @notice 用户签名的下注意图（EIP-712）
     */
    struct BetIntent {
        address user;        // 用户地址
        bytes32 topicId;     // 抽象话题 ID
        uint256 amount;      // 最大下注金额
        uint256 nonce;       // 防重放
        uint256 deadline;    // 截止时间
    }

    // 定义 BetStatus 枚举
    enum BetStatus {
        NONE,
        INTENT_CONSUMED,
        FUNDS_LOCKED,
        EXECUTED,
        SETTLED,
        REFUNDED
    }

    /* ========== Errors ========== */
    error IntentExpired(uint256 deadline, uint256 currentTime);   // 下注意图已过期
    error InvalidNonce(address user, uint256 expected, uint256 provided);   // 非法nonce
    error IntentAlreadyUsed(bytes32 intentHash);    // 下注意图已被使用
    error BetAlreadyExecuted(bytes32 betId);    // 下注已执行
    error InvalidSignature(address recovered, address expected);    // 签名者无效

    /* ========== Events ========== */

    event BetIntentConsumed(
        bytes32 indexed betId,
        address indexed user,
        bytes32 indexed topicId,
        uint256 amount,
        bytes32 intentHash
    );

    function executeBetIntent(BetIntent calldata intent, bytes calldata signature) external;
    function getBetStatus(bytes32 betId) external view returns (BetStatus);
    function updateBetStatus(bytes32 betId, BetStatus status) external;
    function getBetTimestamp(bytes32 betId) external view returns (uint256);

}