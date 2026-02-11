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
 */
contract ProtocolAccessUpgradeable is Initializable, AccessControlUpgradeable {
    bytes32 public constant EXECUTOR_ROLE = keccak256("EXECUTOR_ROLE");
    bytes32 public constant ORACLE_ROLE   = keccak256("ORACLE_ROLE");
    bytes32 public constant GOVERNANCE_ROLE = keccak256("GOVERNANCE_ROLE");
    bytes32 public constant WITHDRAW_ROLE = keccak256("WITHDRAW_ROLE");

    function __ProtocolAccess_init(address admin) internal onlyInitializing {
        __AccessControl_init();
        _grantRole(DEFAULT_ADMIN_ROLE, admin);
        _grantRole(GOVERNANCE_ROLE, admin);
    }
}
