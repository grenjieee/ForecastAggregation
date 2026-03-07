// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "@openzeppelin/contracts-upgradeable/access/AccessControlUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

/**
 * @title ProtocolAccessUpgradeable
 * @notice 提供基础角色权限管理，可作为其他模块继承
 * Roles:
 * - EXECUTOR_ROLE：链下 Bot/执行器
 * - ORACLE_ROLE：结果上报权限
 * - GOVERNANCE_ROLE：协议管理权限
 * - WITHDRAW_ROLE：资金提取权限
 */
contract ProtocolAccessUpgradeable is Initializable, AccessControlUpgradeable {
    bytes32 public constant GOVERNANCE_ROLE = keccak256("GOVERNANCE_ROLE");
    bytes32 public constant EXECUTOR_ROLE = keccak256("EXECUTOR_ROLE");
    bytes32 public constant ORACLE_ROLE   = keccak256("ORACLE_ROLE");
    bytes32 public constant WITHDRAW_ROLE = keccak256("WITHDRAW_ROLE");

    function initialize(address admin) public initializer {
        __AccessControl_init();
        _grantRole(DEFAULT_ADMIN_ROLE, admin);
        _grantRole(GOVERNANCE_ROLE, admin);

        // 设置所有角色的管理者为 GOVERNANCE_ROLE
        _setRoleAdmin(EXECUTOR_ROLE, GOVERNANCE_ROLE);
        _setRoleAdmin(ORACLE_ROLE, GOVERNANCE_ROLE);
        _setRoleAdmin(WITHDRAW_ROLE, GOVERNANCE_ROLE);
    }

    /* ========== 白名单管理函数 ========== */

    /**
     * @notice 添加 EXECUTOR 白名单
     */
    function addExecutor(address account) external onlyRole(GOVERNANCE_ROLE)
    {
        grantRole(EXECUTOR_ROLE, account);
    }

    function removeExecutor(address account) external onlyRole(GOVERNANCE_ROLE)
    {
        revokeRole(EXECUTOR_ROLE, account);
    }

    /**
     * @notice 添加 ORACLE 白名单
     */
    function addOracle(address account) external onlyRole(GOVERNANCE_ROLE)
    {
        grantRole(ORACLE_ROLE, account);
    }

    function removeOracle(address account) external onlyRole(GOVERNANCE_ROLE)
    {
        revokeRole(ORACLE_ROLE, account);
    }

    /**
     * @notice 添加 WITHDRAW 白名单
     */
    function addWithdrawer(address account) external onlyRole(GOVERNANCE_ROLE)
    {
        grantRole(WITHDRAW_ROLE, account);
    }

    function removeWithdrawer(address account) external onlyRole(GOVERNANCE_ROLE)
    {
        revokeRole(WITHDRAW_ROLE, account);
    }

    /* ========== 只读查询 ========== */

    function isExecutor(address account) external view returns (bool)
    {
        return hasRole(EXECUTOR_ROLE, account);
    }

    function isOracle(address account) external view returns (bool)
    {
        return hasRole(ORACLE_ROLE, account);
    }

    function isWithdrawer(address account) external view returns (bool)
    {
        return hasRole(WITHDRAW_ROLE, account);
    }

}
