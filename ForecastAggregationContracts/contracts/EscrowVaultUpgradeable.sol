// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "./lib/ProtocolAccessLib.sol";
import "./interface/IProtocolAccess.sol";
import "./interface/IEscrowVault.sol";
import "./interface/IBetRouter.sol";
import "./interface/ITopicRegistry.sol";
import "./interface/IOracleAdapter.sol";
import "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

/**
 * @title EscrowVaultUpgradeable
 * @notice 托管用户资金，Executor 可以锁定/释放资金
 * - 支持 ERC20
 * - 可升级 + Pause + 权限控制
 */
contract EscrowVaultUpgradeable is ReentrancyGuardUpgradeable, PausableUpgradeable, UUPSUpgradeable, IEscrowVault
{
    using SafeERC20Upgradeable for IERC20Upgradeable;
    using ProtocolAccessLib for IProtocolAccess;

    IERC20Upgradeable public token; // 托管 ERC20

    address private betRouterAddress;
    // address private topicRegistryAddress;
    // address private oracleAdapterAddress;
    IProtocolAccess public accessManager;

    mapping(bytes32 => uint256) public lockedAmount; // betId → 金额

    uint256[50] private __gap;

    /* ========== Initializer ========== */

    function initialize(address _accessManager,
        IERC20Upgradeable _token,
        address _betRouterAddress
        // address _topicRegistryAddress,
        // address _oracleAdapterAddress
        ) 
        external 
        initializer 
    {
        require(_accessManager != address(0), "Invalid access manager");
        accessManager = IProtocolAccess(_accessManager);
        __ReentrancyGuard_init();
        __Pausable_init();
        __UUPSUpgradeable_init();

        token = _token;
        betRouterAddress = _betRouterAddress;
        // topicRegistryAddress = _topicRegistryAddress;
        // oracleAdapterAddress = _oracleAdapterAddress;
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

    modifier judgeExpiredLocked(bytes32 _betId) {

        uint256 betFundsLockTime = IBetRouter(betRouterAddress).getBetTimestamp(_betId) + 1 hours; // 1小时超时时间

        if (block.timestamp > betFundsLockTime) {
            revert ExpiredLocked(_betId, block.timestamp, betFundsLockTime);
        }

        _;
    }

    modifier judgeReleaseFunds(bytes32 _betId, uint256 _amount) {

        if (lockedAmount[_betId] < _amount || token.balanceOf(address(this)) < _amount) {
            revert InsufficientLocked(_betId, lockedAmount[_betId], _amount);
        }

        _;
    }

    modifier judgeExecutedFunds(bytes32 _betId, bytes32 _topicId) {

        IBetRouter betRouter = IBetRouter(betRouterAddress);
        // ITopicRegistry topicRegistry = ITopicRegistry(topicRegistryAddress);
        // IOracleAdapter oracle = IOracleAdapter(oracleAdapterAddress);

        if (betRouter.getBetStatus(_betId) != IBetRouter.BetStatus.FUNDS_LOCKED 
            // || topicRegistry.getTopicActive(_topicId) 
            // || !oracle.getTopicResolvedActive(_topicId)
            ) 
        {
            revert CannotExecuteFunds(_betId, _topicId);
        }

        _;
    }

    /* ========== GOVERNANCE_ROLE ========== */

    function setBetRouterAddress(address _betRouterAddress) external onlyGovernance {
        betRouterAddress = _betRouterAddress;
    }

    // 暂时不提供修改 TopicRegistry 和 OracleAdapter 地址的接口，因为它们目前没有实际用途，后续如果需要再添加

    // function setTopicRegistryAddress(address _topicRegistryAddress) external onlyGovernance {
    //     topicRegistryAddress = _topicRegistryAddress;
    // }

    // function setOracleAdapterAddress(address _oracleAdapterAddress) external onlyGovernance {
    //     oracleAdapterAddress = _oracleAdapterAddress;
    // }

    /* ========== VIEW ========== */

    function getBetRouterAddress() external view returns (address) {
        return betRouterAddress;
    }

    // 暂时不提供获取 TopicRegistry 和 OracleAdapter 地址的接口，因为它们目前没有实际用途，后续如果需要再添加

    // function getTopicRegistryAddress() external view returns (address) {
    //     return topicRegistryAddress;
    // }

    // function getOracleAdapterAddress() external view returns (address) {
    //     return oracleAdapterAddress;
    // }    

    /* ========== Internal Logic ========== */

    function _updateBetStatus(bytes32 _betId, IBetRouter.BetStatus _status) internal {
        IBetRouter betRouter = IBetRouter(betRouterAddress);
        betRouter.updateBetStatus(_betId, _status);
    }

    function _updateBetStatusWithSig(bytes32 _betId, IBetRouter.BetStatus _status, bytes calldata signature) internal {
        IBetRouter betRouter = IBetRouter(betRouterAddress);
        betRouter.updateBetStatusWithSig(_betId, _status, signature);
    }


    /* ========== Core Logic ========== */

    /**
     * @notice 锁定该betId所需的资金,并且更新状态机到FUNDS_LOCKED，该动作前需要ERC20的授权
     * @param betId Bet ID
     * @param amount 锁定金额
     * 
     * Emits {FundsLocked}
     */
    function lockFunds(bytes32 betId, uint256 amount, bytes calldata signature)
        external
        override
        whenNotPaused
        judgeExpiredLocked(betId)
    {   
        token.safeTransferFrom(msg.sender, address(this), amount);
        lockedAmount[betId] += amount;
        _updateBetStatusWithSig(betId, IBetRouter.BetStatus.FUNDS_LOCKED, signature);
        emit FundsLocked(betId, msg.sender, amount);
    }

    /**
     * @notice 释放对应betId的资金(退款),并且更新状态机到REFUNDED
     * @param betId Bet ID
     * @param to 释放到的钱包地址
     * @param amount 释放金额
     */
    function releaseFunds(bytes32 betId, address to, uint256 amount, bytes calldata signature)
        external
        override
        whenNotPaused
        judgeReleaseFunds(betId, amount)
        nonReentrant
    {        
        lockedAmount[betId] -= amount;
        token.safeTransfer(to, amount);
        _updateBetStatusWithSig(betId, IBetRouter.BetStatus.REFUNDED, signature);
        emit FundsReleased(betId, to, amount);
    }

    /**
     * @notice Executor 释放对应betId的资金到指定下注专用钱包(执行预测市场话题下注),并且更新状态机到EXECUTED.记得需要先将后端下注钱包添加Executor
     * @param betId Bet ID
     * @param topicId Topic ID
     * @param to 释放到的钱包地址,只能是后端使用去预测市场执行下注的钱包
     * @param amount 释放金额
     */
    function executedFunds(bytes32 betId, bytes32 topicId, address to, uint256 amount)
        external
        override
        whenNotPaused
        judgeExecutedFunds(betId, topicId)
        judgeReleaseFunds(betId, amount)
        nonReentrant
        onlyExecutor
    {        
        lockedAmount[betId] -= amount;
        token.safeTransfer(to, amount);
        _updateBetStatus(betId, IBetRouter.BetStatus.EXECUTED);
        emit BetExecuted(betId);
    }

    /* ========== Pause Control ========== */

    function pause() external onlyGovernance { _pause(); }
    function unpause() external onlyGovernance { _unpause(); }

    /* ========== Upgrade Control ========== */
    
    function _authorizeUpgrade(address) internal override onlyGovernance {}
}
