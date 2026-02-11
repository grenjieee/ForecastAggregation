// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "./interface/IBetRouter.sol";
import "./interface/ITopicRegistry.sol";
import "./ProtocolAccessUpgradeable.sol";
import "./EscrowVaultUpgradeable.sol";
import "./OracleAdapterUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

/**
 * @title SettlementUpgradeable
 * @notice 处理结算逻辑、抽佣
 */
contract SettlementUpgradeable is ProtocolAccessUpgradeable, PausableUpgradeable, UUPSUpgradeable
{
    EscrowVaultUpgradeable public vault;
    OracleAdapterUpgradeable public oracle;
    address private betRouterAddress;
    address private topicRegistryAddress;

    uint256 public feeRate; // 10000 = 100%
    address public feeVault;

    error BetStatusIsNotEXECUTED(bytes32 betId);
    error TopicIsNotEnd(bytes32 topicId);
    error TopicIsNotResolved(bytes32 topicId);

    event Settled(bytes32 indexed betId, uint256 payout, uint256 fee);

    /* ========== Initializer ========== */

    function initialize(
        address admin,
        EscrowVaultUpgradeable _vault,
        OracleAdapterUpgradeable _oracle,
        address _feeVault,
        uint256 _feeRate,
        address _betRouterAddress,
        address _topicRegistryAddress
    ) external initializer {
        __ProtocolAccess_init(admin);
        __Pausable_init();
        __UUPSUpgradeable_init();

        vault = _vault;
        oracle = _oracle;
        feeVault = _feeVault;
        feeRate = _feeRate;
        betRouterAddress = _betRouterAddress;
        topicRegistryAddress = _topicRegistryAddress;
    }

    function setBetRouterAddress(address _betRouterAddress) external onlyRole(GOVERNANCE_ROLE) {
        betRouterAddress = _betRouterAddress;
    }

    function getBetRouterAddress() external view returns (address) {
        return betRouterAddress;
    }

    function setTopicRegistryAddress(address _topicRegistryAddress) external onlyRole(GOVERNANCE_ROLE) {
        topicRegistryAddress = _topicRegistryAddress;
    }

    function getTopicRegistryAddress() external view returns (address) {
        return topicRegistryAddress;
    }

    function updateBetStatus(bytes32 _betId, IBetRouter.BetStatus _status) internal {
        IBetRouter betRouter = IBetRouter(betRouterAddress);
        betRouter.updateBetStatus(_betId, _status);
    }

    function getCurrentBetStatus(bytes32 _betId) internal view returns (IBetRouter.BetStatus){
        IBetRouter betRouter = IBetRouter(betRouterAddress);
        return betRouter.getBetStatus(_betId);
    }

    function getCurrentTopicActive(bytes32 _topicId) internal view returns (bool){
        ITopicRegistry topicRegistry = ITopicRegistry(topicRegistryAddress);
        return topicRegistry.getTopicActive(_topicId);
    }

    /* ========== modifier ========== */

    /**
     * @notice 判断是否可以结算(只有BetStatus状态机是EXECUTED时,TopicId对应的话题已结束,TopicId对应的话题已上报结果时才可以结算)
     * @param _betId 下注ID
     * @param _topicId TopicID
    **/
    modifier judgeSettle(bytes32 _betId, bytes32 _topicId) {
        IBetRouter.BetStatus currentBetStatus = getCurrentBetStatus(_betId);

        if (currentBetStatus != IBetRouter.BetStatus.EXECUTED) {
            revert BetStatusIsNotEXECUTED(_betId);
        }

        if (getCurrentTopicActive(_topicId)) {
            revert TopicIsNotEnd(_topicId);
        }

        if (!oracle.getTopicResolvedActive(_topicId)) {
            revert TopicIsNotResolved(_topicId);
        }

        _;
    }

    /* ========== Core Logic ========== */

    /**
     * @notice 结算胜利的下注
     * @param _betId 下注ID
     * @param _topicId TopicID
     * @param user 用户地址
     * @param principal 本金
     * @param payout 总回报金额
     * 
     * Emits {Settled}
     */
    function settleWin(bytes32 _betId, bytes32 _topicId, address user, uint256 principal, uint256 payout)
        external
        onlyRole(EXECUTOR_ROLE)
        whenNotPaused
        judgeSettle(_betId, _topicId)
    {   
        uint256 profit = payout > principal ? payout - principal : 0;
        uint256 fee = (profit * feeRate) / 10000;

        vault.releaseFunds(_betId, feeVault, fee);
        vault.releaseFunds(_betId, user, payout - fee);

        updateBetStatus(_betId, IBetRouter.BetStatus.SETTLED);

        emit Settled(_betId, payout, fee);
    }

    /**
     * @notice 结算失败的下注(Executor去进行下注的时候,进行将对应的本金进行执行取出)
     * @param _betId 下注ID
     * @param _topicId TopicID
     * 
     * Emits {Settled}
     */
    function settleLoss(bytes32 _betId, bytes32 _topicId) external onlyRole(EXECUTOR_ROLE) whenNotPaused judgeSettle(_betId, _topicId) {

        IBetRouter.BetStatus currentBetStatus = getCurrentBetStatus(_betId);

        if (currentBetStatus != IBetRouter.BetStatus.EXECUTED) {
            revert BetStatusIsNotEXECUTED(_betId);
        }

        updateBetStatus(_betId, IBetRouter.BetStatus.SETTLED);

        // 本金已在 Vault，Executor 可以处理无需释放
        emit Settled(_betId, 0, 0);
    }

    /* ========== Pause Control ========== */

    function pause() external onlyRole(GOVERNANCE_ROLE) { _pause(); }
    function unpause() external onlyRole(GOVERNANCE_ROLE) { _unpause(); }

    /* ========== Upgrade Control ========== */

    function _authorizeUpgrade(address) internal override onlyRole(GOVERNANCE_ROLE) {}
}
