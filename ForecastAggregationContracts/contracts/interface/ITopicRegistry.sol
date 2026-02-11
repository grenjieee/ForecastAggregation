// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

interface ITopicRegistry {

    struct Topic {
        string canonicalQuestion; // 标准化问题
        uint8 outcomeCount;       // 结果数量
        bool active;              // 是否可用
    }

    error TopicExists(bytes32 topicId); // Topic已存在

    event TopicCreated(bytes32 indexed topicId, string question);
    event TopicStatusUpdated(bytes32 indexed topicId, bool active);

    function createTopic(bytes32 topicId, string calldata question, uint8 outcomeCount) external;   // 创建Topic
    function setTopicActive(bytes32 topicId, bool active) external; // 设置Topic可用状态
    function getTopicActive(bytes32 topicId) external view returns (bool);  // 获取Topic可用状态
}