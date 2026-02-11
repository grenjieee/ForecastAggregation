// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

interface IEscrowVault {

    error ExpiredLocked(bytes32 betId, uint256 currentTime, uint256 lockTimestamp); // lockFunds超时检测
    error InsufficientLocked(bytes32 betId, uint256 lockedAmount, uint256 requiredAmount); // 释放资金时锁定金额不足
    error CannotExecuteFunds(bytes32 betId, bytes32 topicId); // 获取执行资金时,不满足执行条件

    event FundsLocked(bytes32 indexed betId, address from, uint256 amount);
    event FundsReleased(bytes32 indexed betId, address to, uint256 amount);
    event BetExecuted(bytes32 indexed betId);

    function lockFunds(bytes32 betId, uint256 amount) external;   // 锁定用户资金
    function releaseFunds(bytes32 betId, address to, uint256 amount) external;  // 退还用户资金
    function executedFunds(bytes32 betId, bytes32 topicId, address to, uint256 amount) external;  // 获取对应betId的执行资金
}