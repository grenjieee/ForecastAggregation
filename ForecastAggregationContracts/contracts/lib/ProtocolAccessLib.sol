// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "../interface/IProtocolAccess.sol";

library ProtocolAccessLib {

    error NotGovernanceRole(address caller);  // 不是ProtocolAccessUpgradeable合约中的Governance角色
    error NotExecutorRole(address caller);    // 不是ProtocolAccessUpgradeable合约中的Executor角色
    error NotOracleRole(address caller);      // 不是ProtocolAccessUpgradeable合约中的Oracle角色
    error NotWithdrawRole(address caller);    // 不是ProtocolAccessUpgradeable合约中的Withdraw角色

    function enforceGovernance(IProtocolAccess access) internal view {
        if (
            !access.hasRole(
                access.GOVERNANCE_ROLE(),
                msg.sender
            )
        ) revert NotGovernanceRole(msg.sender);
    }

    function enforceExecutor(IProtocolAccess access) internal view {
        if (
            !access.hasRole(
                access.EXECUTOR_ROLE(),
                msg.sender
            )
        ) revert NotExecutorRole(msg.sender);
    }

    function enforceOracle(IProtocolAccess access) internal view {
        if (
            !access.hasRole(
                access.ORACLE_ROLE(),
                msg.sender
            )
        ) revert NotOracleRole(msg.sender);
    }

    function enforceWithdrawRole(IProtocolAccess access) internal view {
        if (
            !access.hasRole(
                access.WITHDRAW_ROLE(),
                msg.sender
            )
        ) revert NotWithdrawRole(msg.sender);
    }
}