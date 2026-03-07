// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

interface IProtocolAccess {
    function GOVERNANCE_ROLE() external view returns (bytes32);
    function EXECUTOR_ROLE() external view returns (bytes32);    
    function ORACLE_ROLE() external view returns (bytes32);
    function WITHDRAW_ROLE() external view returns (bytes32);

    function hasRole(bytes32 role, address account) external view returns (bool);
    function isExecutor(address account) external view returns (bool);
}