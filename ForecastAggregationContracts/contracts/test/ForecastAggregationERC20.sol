// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/token/ERC20/ERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/ERC20BurnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

/**
 * @title ForecastAggregationERC20
 * @notice 标准可升级 ERC20 实现，符合 IERC20Upgradeable 规范
 */
contract ForecastAggregationERC20 is
    Initializable,
    ERC20Upgradeable,
    ERC20BurnableUpgradeable,
    OwnableUpgradeable
{
    /// @custom:oz-upgrades-unsafe-allow constructor
    // constructor() {
    //     _disableInitializers();
    // }

    /**
     * @notice 初始化函数（替代 constructor）
     * @param _name 代币名称
     * @param _symbol 代币符号
     * @param initialSupply 初始铸造数量（最小单位）
     * @param _owner 合约拥有者
     **/
    function initialize(
        string memory _name,
        string memory _symbol,
        uint256 initialSupply,
        address _owner
    ) public initializer {
        __ERC20_init(_name, _symbol);
        __ERC20Burnable_init();
        __Ownable_init();

        if (initialSupply > 0) {
            _mint(_owner, initialSupply);
        }
    }

    /**
     * @notice 仅 owner 可铸造
     */
    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    /**
     * @notice 可自定义 decimals
     */
    function decimals() public pure override returns (uint8) {
        return 6;
    }

    // ========= Storage Gap（升级安全） =========
    uint256[50] private __gap;
}
