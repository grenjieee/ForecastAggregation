// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "./lib/ProtocolAccessLib.sol";
import "./interface/IProtocolAccess.sol";
import "./interface/ITopicRegistry.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

/**
 * @title TopicRegistryUpgradeable
 * @notice 管理跨预测市场的 Topic 映射
 * - 可升级 UUPS
 * - 可 Pause
 */
contract TopicRegistryUpgradeable is PausableUpgradeable, UUPSUpgradeable, ITopicRegistry
{
    using ProtocolAccessLib for IProtocolAccess;

    mapping(bytes32 => Topic) public topics;

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
     * @notice 创建Topic,确保合约中有对应的预测话题
     * @param topicId TopicID,由后端生成(相当平台整合多预测市场的同一话题的统一ID)
     * @param question 标准化问题
     * @param outcomeCount 结果数量
     * 
     * Emits {TopicCreated}
    **/
    function createTopic(
        bytes32 topicId,
        string calldata question,
        uint8 outcomeCount
    ) external override onlyOracle whenNotPaused {
        if (topics[topicId].outcomeCount != 0) {
            revert TopicExists(topicId);
        }

        topics[topicId] = Topic({
            canonicalQuestion: question,
            outcomeCount: outcomeCount,
            active: true
        });

        emit TopicCreated(topicId, question);
    }

    /* ========== EXECUTOR_ROLE ========== */

    /**
     * @notice 设置Topic状态(用于平台上架Topic发现问题时使用,以及预测话题已出现结果后下架)
     * @param topicId Topic ID
     * @param active Topic状态
     * 
     * Emits {TopicStatusUpdated}
    **/
    function setTopicActive(bytes32 topicId, bool active)
        external override onlyOracle whenNotPaused {
        topics[topicId].active = active;
        emit TopicStatusUpdated(topicId, active);
    }

    /* ========== View ========== */

    /**
     * @notice 获取Topic状态
     * @param topicId Topic ID
     * @return bool 对应Topic状态
    **/
    function getTopicActive(bytes32 topicId) external view override returns (bool){
        return topics[topicId].active;
    }

    /* ========== Pause Control ========== */

    function pause() external onlyGovernance { _pause(); }
    function unpause() external onlyGovernance { _unpause(); }

    /* ========== Upgrade Control ========== */
    
    function _authorizeUpgrade(address) internal override onlyGovernance {}
}
