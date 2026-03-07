// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "./lib/ProtocolAccessLib.sol";
import "./interface/IProtocolAccess.sol";
import "./interface/IBetRouter.sol";
import "./interface/ITopicRegistry.sol";
import "./EscrowVaultUpgradeable.sol";
import "./OracleAdapterUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

/**
 * @title SettlementUpgradeable
 * @notice 处理结算逻辑、抽佣
 */
contract SettlementUpgradeable is PausableUpgradeable, UUPSUpgradeable
{   
    using ProtocolAccessLib for IProtocolAccess;

    EscrowVaultUpgradeable public vault;
    // OracleAdapterUpgradeable public oracle;
    address private betRouterAddress;
    // address private topicRegistryAddress;
    IProtocolAccess public accessManager;

    uint256 public feeRate; // 10000 = 100%
    address public feeVault;

    uint256[50] private __gap;

    error BetStatusIsNotEXECUTED(bytes32 betId);
    error TopicIsNotEnd(bytes32 topicId);
    error TopicIsNotResolved(bytes32 topicId);

    event Settled(bytes32 indexed betId, uint256 payout, uint256 fee);

    /* ========== Initializer ========== */

    function initialize(
        address _accessManager,
        EscrowVaultUpgradeable _vault,
        // OracleAdapterUpgradeable _oracle,
        address _feeVault,
        uint256 _feeRate,
        address _betRouterAddress
        // address _topicRegistryAddress
    ) external initializer {
        require(_accessManager != address(0), "Invalid access manager");
        accessManager = IProtocolAccess(_accessManager);

        __Pausable_init();
        __UUPSUpgradeable_init();

        vault = _vault;
        // oracle = _oracle;
        feeVault = _feeVault;
        feeRate = _feeRate;
        betRouterAddress = _betRouterAddress;
        // topicRegistryAddress = _topicRegistryAddress;
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


    // /**
    //  * @notice 判断是否可以结算(只有BetStatus状态机是EXECUTED时,TopicId对应的话题已结束,TopicId对应的话题已上报结果时才可以结算)
    //  * @param _betId 下注ID
    //  * @param _topicId TopicID
    // **/
    // modifier judgeSettle(bytes32 _betId, bytes32 _topicId) {
    //     IBetRouter.BetStatus currentBetStatus = getCurrentBetStatus(_betId);

    //     if (currentBetStatus != IBetRouter.BetStatus.EXECUTED) {
    //         revert BetStatusIsNotEXECUTED(_betId);
    //     }

    //     if (getCurrentTopicActive(_topicId)) {
    //         revert TopicIsNotEnd(_topicId);
    //     }

    //     if (!oracle.getTopicResolvedActive(_topicId)) {
    //         revert TopicIsNotResolved(_topicId);
    //     }

    //     _;
    // }

    function setBetRouterAddress(address _betRouterAddress) external onlyGovernance {
        betRouterAddress = _betRouterAddress;
    }

    function getBetRouterAddress() external view returns (address) {
        return betRouterAddress;
    }

    // function setTopicRegistryAddress(address _topicRegistryAddress) external onlyGovernance {
    //     topicRegistryAddress = _topicRegistryAddress;
    // }

    // function getTopicRegistryAddress() external view returns (address) {
    //     return topicRegistryAddress;
    // }

    function updateBetStatus(bytes32 _betId, IBetRouter.BetStatus _status, bytes calldata signature) internal {
        IBetRouter betRouter = IBetRouter(betRouterAddress);
        betRouter.updateBetStatusWithSig(_betId, _status, signature);
    }

    function getCurrentBetStatus(bytes32 _betId) internal view returns (IBetRouter.BetStatus){
        IBetRouter betRouter = IBetRouter(betRouterAddress);
        return betRouter.getBetStatus(_betId);
    }

    // function getCurrentTopicActive(bytes32 _topicId) internal view returns (bool){
    //     ITopicRegistry topicRegistry = ITopicRegistry(topicRegistryAddress);
    //     return topicRegistry.getTopicActive(_topicId);
    // }

    /* ========== Core Logic ========== */

    /**
     * @notice 结算胜利的下注
     * @param _betId 下注ID
     * @param user 用户地址
     * @param principal 本金
     * @param payout 总回报金额
     * 
     * Emits {Settled}
     */
    function settleWin(bytes32 _betId, address user, uint256 principal, uint256 payout, bytes calldata signature_refund, bytes calldata signature_settle)
        external
        whenNotPaused
    {   
        uint256 profit = payout > principal ? payout - principal : 0;
        uint256 fee = (profit * feeRate) / 10000;

        vault.releaseFunds(_betId, feeVault, fee, signature_refund);
        vault.releaseFunds(_betId, user, payout - fee, signature_refund);

        updateBetStatus(_betId, IBetRouter.BetStatus.SETTLED, signature_settle);

        emit Settled(_betId, payout, fee);
    }

    /**
     * @notice 结算失败的下注(Executor去进行下注的时候,进行将对应的本金进行执行取出)
     * @param _betId 下注ID
     * 
     * Emits {Settled}
     */
    function settleLoss(bytes32 _betId, bytes calldata signature) external whenNotPaused {

        IBetRouter.BetStatus currentBetStatus = getCurrentBetStatus(_betId);

        if (currentBetStatus != IBetRouter.BetStatus.EXECUTED) {
            revert BetStatusIsNotEXECUTED(_betId);
        }

        updateBetStatus(_betId, IBetRouter.BetStatus.SETTLED, signature);

        // 本金已在 Vault，Executor 可以处理无需释放
        emit Settled(_betId, 0, 0);
    }

    /* ========== Pause Control ========== */

    function pause() external onlyGovernance { _pause(); }
    function unpause() external onlyGovernance { _unpause(); }

    /* ========== Upgrade Control ========== */

    function _authorizeUpgrade(address) internal override onlyGovernance {}
}
