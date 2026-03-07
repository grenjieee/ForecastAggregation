// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "./lib/ProtocolAccessLib.sol";
import "./interface/IProtocolAccess.sol";
import "./interface/IOracleAdapter.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

/**
 * @title OracleAdapterUpgradeable
 * @notice 接收外部预测市场结果
 */
contract OracleAdapterUpgradeable is PausableUpgradeable, UUPSUpgradeable, IOracleAdapter
{   
    using ProtocolAccessLib for IProtocolAccess;

    mapping(bytes32 => uint8) public results; // topicId → outcome
    mapping(bytes32 => bool) public resolved; // topicId 是否已上报

    IProtocolAccess public accessManager;
    uint256[50] private __gap;

    /* ========== Initializer ========== */

    function initialize(address _accessManager) external initializer {
        require(_accessManager != address(0), "Invalid access manager");
        accessManager = IProtocolAccess(_accessManager);
        __Pausable_init();
        __UUPSUpgradeable_init();
    }

    /* ========== modifier ========== */

    modifier onlyGovernance() {
        accessManager.enforceGovernance();
        _;
    }

    modifier onlyOracle() {
        accessManager.enforceOracle();
        _;
    }
    
    /* ========== Core Logic ========== */

    /**
     * @notice 上报Topic的实际结果,确保合约中有对应的预测话题
     * @param topicId Topic ID
     * @param outcome 标准化问题结果
     * 
     * Emits {ResultReported}
    **/
    function reportResult(bytes32 topicId, uint8 outcome)
        external
        override
        onlyOracle
        whenNotPaused
    {
        if (resolved[topicId]) {
            revert AlreadyResolved(topicId); 
        }
        resolved[topicId] = true;
        results[topicId] = outcome;
        emit ResultReported(topicId, outcome);
    }

    /* ========== VIEW ========== */

    /**
     * @notice 获取Topic实际结果的上报状态
     * @param topicId TopicID
     * @return bool 对应Topic实际结果是否已上报
    **/
    function getTopicResolvedActive(bytes32 topicId) external view override returns (bool){
        return resolved[topicId];
    }

    /**
     * @notice 获取Topic实际结果
     * @param topicId TopicID
     * @return uint8 对应Topic实际结果
    **/
    function getTopicResult(bytes32 topicId) external view override returns (uint8) {
        return results[topicId];
    }

    /* ========== Pause Control ========== */

    function pause() external onlyGovernance { _pause(); }
    function unpause() external onlyGovernance { _unpause(); }

    /* ========== Upgrade Control ========== */

    function _authorizeUpgrade(address) internal override onlyGovernance {}
}
