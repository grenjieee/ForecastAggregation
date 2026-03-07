// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import "./lib/ProtocolAccessLib.sol";
import "./interface/IProtocolAccess.sol";
import "@openzeppelin/contracts-upgradeable/access/AccessControlUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

/**
 * @title FeeVault
 * @notice 管理手续费的提现操作
 */
contract FeeVault is PausableUpgradeable, UUPSUpgradeable {
    using SafeERC20Upgradeable for IERC20Upgradeable;
    using ProtocolAccessLib for IProtocolAccess;

    IERC20Upgradeable public token; // 手续费代币
    IProtocolAccess public accessManager;
    uint256[50] private __gap;
    
    error InsufficientFeeBalance(uint256 balance, uint256 amount);

    event FeeWithdrawn(address indexed to, uint256 amount);

    /* ========== Initializer ========== */

    /**
     * @notice 初始化合约
     * @param _accessManager 管理员地址
     * @param _token 手续费代币地址
     */
    function initialize(address _accessManager, IERC20Upgradeable _token) external initializer {
        require(_accessManager != address(0), "Invalid access manager");
        accessManager = IProtocolAccess(_accessManager);
        __Pausable_init();
        __UUPSUpgradeable_init();

        token = _token;
    }

    /* ========== modifier ========== */

    modifier onlyGovernance() {
        accessManager.enforceGovernance();
        _;
    }

    modifier onlyWithdrawRole() {
        accessManager.enforceWithdrawRole();
        _;
    }

    /* ========== Core Logic ========== */

    /**
     * @notice 提现手续费（仅限 WITHDRAW_ROLE 角色调用）
     * @param to 提现目标地址
     * @param amount 提现金额
     * 
     * Emits {FeeWithdrawn}
     */
    function withdrawFee(address to, uint256 amount) external onlyWithdrawRole whenNotPaused {
        require(amount > 0, "FeeVault: amount must be greater than zero");
        uint256 balance = token.balanceOf(address(this));
        if (balance < amount) {
            revert InsufficientFeeBalance(balance, amount);
        }

        token.safeTransfer(to, amount);
        emit FeeWithdrawn(to, amount);
    }

    /**
     * @notice 查询当前手续费余额
     * @return 手续费余额
     */
    function getFeeBalance() external view returns (uint256) {
        return token.balanceOf(address(this));
    }

    /* ========== Pause Control ========== */

    function pause() external onlyGovernance {
        _pause();
    }

    function unpause() external onlyGovernance {
        _unpause();
    }

    /* ========== Upgrade Control ========== */

    function _authorizeUpgrade(address newImplementation) internal override onlyGovernance {}
}