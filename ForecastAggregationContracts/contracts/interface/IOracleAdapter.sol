// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

interface IOracleAdapter {

    error AlreadyResolved(bytes32 topicId); // topicId对应的话题结果已经被上报

    event ResultReported(bytes32 indexed topicId, uint8 outcome);

    function reportResult(bytes32 topicId, uint8 outcome) external; // 上报Topic的实际结果
    function getTopicResolvedActive(bytes32 topicId) external view returns (bool);  // 获取Topic实际结果的上报状态
    function getTopicResult(bytes32 topicId) external view returns (uint8); // 获取Topic实际结果
}