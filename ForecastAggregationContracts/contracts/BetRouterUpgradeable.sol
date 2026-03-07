// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

/* ========= OpenZeppelin Upgradeable ========= */
import "./lib/ProtocolAccessLib.sol";
import "./interface/IProtocolAccess.sol";
import "./interface/IBetRouter.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/cryptography/EIP712Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/cryptography/ECDSAUpgradeable.sol";

/**
 * @title BetRouterUpgradeable
 * @notice 预测市场聚合协议的用户入口合约
 *         - EIP-712 Intent 校验
 *         - 可升级（UUPS）
 *         - 可暂停
 *         - 权限化执行
 */
contract BetRouterUpgradeable is Initializable, UUPSUpgradeable, PausableUpgradeable, EIP712Upgradeable, IBetRouter
{
    using ECDSAUpgradeable for bytes32;
    using ProtocolAccessLib for IProtocolAccess;

    bytes32 public constant BET_INTENT_TYPEHASH =
        keccak256(
            "BetIntent(address user,bytes32 topicId,uint256 amount,uint256 nonce,uint256 deadline)"
        );

    /* ========== Storage ========== */

    mapping(address => uint256) public nonces;  // 用户 nonce
    mapping(bytes32 => bool) public usedIntents;    // Intent 是否已使用
    mapping(bytes32 => BetStatus) public betStatus; // betId → 状态
    mapping(bytes32 => uint256) public betTimestamp;    // betId → 时间戳
    IProtocolAccess public accessManager;
    uint256[50] private __gap;

    /* ========== Initializer ========== */

    function initialize(address _accessManager) external initializer {
        require(_accessManager != address(0), "Invalid access manager");
        accessManager = IProtocolAccess(_accessManager);
        __Pausable_init();
        __UUPSUpgradeable_init();
        __EIP712_init(
            "PredictionMarketAggregator",
            "1"
        );
    }

    /* ========== modifier ========== */

    modifier onlyGovernance() {
        accessManager.enforceGovernance();
        _;
    }

    modifier onlyExecutor() {
        accessManager.enforceExecutor();
        _;
    }

    modifier onlyOracle() {
        accessManager.enforceOracle();
        _;
    }

    modifier onlyWithdrawRole() {
        accessManager.enforceWithdrawRole();
        _;
    }

    /* ========== Core Logic ========== */

    /**
     * @notice 执行用户下注 Intent（只能由 Executor 调用）
     * @dev 实际“下单”发生在链下，本函数只负责：
     *      - 校验用户授权
     *      - 锁定 Intent
     */
    function executeBetIntent(BetIntent calldata intent, bytes calldata signature)
        external
        override
        whenNotPaused
    {
        if (block.timestamp > intent.deadline) {
            revert IntentExpired(intent.deadline, block.timestamp);
        }

        if (intent.nonce != nonces[intent.user]) {
            revert InvalidNonce(intent.user, nonces[intent.user], intent.nonce);
        }

        bytes32 betId = computeBetId(intent.user, intent.topicId, intent.nonce);

        bytes32 digest = _hashTypedDataV4(
            keccak256(
                abi.encode(
                    BET_INTENT_TYPEHASH,
                    intent.user,
                    intent.topicId,
                    intent.amount,
                    intent.nonce,
                    intent.deadline
                )
            )
        );

        if (usedIntents[digest]) {
            revert IntentAlreadyUsed(digest);
        }

        if (betStatus[betId] != BetStatus.NONE){
            revert BetAlreadyExecuted(betId);
        }

        address signer = digest.recover(signature);
        if (signer != intent.user) {
            revert InvalidSignature(signer, intent.user);
        }
        // 标记 intent 已消费
        usedIntents[digest] = true;
        nonces[intent.user]++;

        betStatus[betId] = BetStatus.INTENT_CONSUMED;
        betTimestamp[betId] = block.timestamp;

        // 触发事件,记录 intent 消费，供链下监听,监听到后由后端去执行下注操作
        emit BetIntentConsumed(
            betId,
            intent.user,
            intent.topicId,
            intent.amount,
            digest
        );

        /**
         * ⚠️ 注意：
         * - 这里不转账
         * - 不关心具体下注市场
         * - 只做“授权消费”
         *
         * 后续步骤：
         * - Executor 用 intent.amount 去 EscrowVault 锁资金
         * - 在 Polymarket / Kalshi 下单
         */
    }

    function computeBetId(address user, bytes32 topicId, uint256 nonce) internal pure returns (bytes32) {
        return keccak256(abi.encode(user, topicId, nonce));
    }

    function getBetStatus(bytes32 betId) external view override returns (BetStatus) {
        return betStatus[betId];
    }

    function updateBetStatus(bytes32 betId, BetStatus status) external override onlyExecutor {
        betStatus[betId] = status;
    }

    /**
     * @notice 安全更新 BetStatus
     * @param betId 下注 ID
     * @param status 新状态
     * @param signature 前端签名 (签名 digest = keccak256(betId, status, signerNonce))  确保前端的钱包已经有Executor角色
     */
    function updateBetStatusWithSig(bytes32 betId, BetStatus status, bytes calldata signature) external override {
        uint256 signerNonce = nonces[tx.origin];
        bytes32 digest = keccak256(abi.encodePacked(betId, status, signerNonce));

        if (usedIntents[digest]) {
            revert IntentAlreadyUsed(digest);
        }

        address signer = ECDSAUpgradeable.recover(digest, signature);
        require(accessManager.isExecutor(signer), "Unauthorized signer");        

        betStatus[betId] = status;
    }

    function getBetTimestamp(bytes32 betId) external view override returns (uint256) {
        return betTimestamp[betId];
    }

    /* ========== Pause Control ========== */

    function pause() external onlyGovernance {
        _pause();
    }

    function unpause() external onlyGovernance {
        _unpause();
    }

    /* ========== Upgrade Control ========== */

    function _authorizeUpgrade(address)
        internal
        override
        onlyGovernance
    {}
}
