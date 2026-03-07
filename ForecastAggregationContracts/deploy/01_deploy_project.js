// * 引入Hardhat的部署和升级工具
const { ethers, deployments, upgrades } = require("hardhat");

// * 引入Node.js内置模块,用于文件操作
const fs = require('fs');
const path = require('path');

module.exports = async ({ getNamedAccounts, deployments }) => {
    // * 从deployments中提取save方法,用于保存部署信息到hardhat-deploy的部署记录
    const { save } = deployments;

    // * 获取预定义的部署账户
    const { deployer } = await getNamedAccounts();

    console.log("所有合约部署的用户地址：", deployer);

    const ForecastAggregationERC20Object = await ethers.getContractFactory("ForecastAggregationERC20");
    const BetRouterUpgradeableObject = await ethers.getContractFactory("BetRouterUpgradeable");
    const EscrowVaultUpgradeableObject = await ethers.getContractFactory("EscrowVaultUpgradeable");
    const FeeVaultObject = await ethers.getContractFactory("FeeVault");
    const OracleAdapterUpgradeableObject = await ethers.getContractFactory("OracleAdapterUpgradeable");
    const ProtocolAccessUpgradeableObject = await ethers.getContractFactory("ProtocolAccessUpgradeable");
    const SettlementUpgradeableObject = await ethers.getContractFactory("SettlementUpgradeable");
    const TopicRegistryUpgradeableObject = await ethers.getContractFactory("TopicRegistryUpgradeable");

    // ? 1.部署ForecastAggregationERC20.sol|BetRouterUpgradeable.sol|EscrowVaultUpgradeable.sol|FeeVault.sol|OracleAdapterUpgradeable.sol|ProtocolAccessUpgradeable.sol|SettlementUpgradeable.sol|TopicRegistryUpgradeable.sol合约

    // 通过代理合约部署
    // 通过OpenZeppelin Hardhat Upgrades插件部署可升级代理合约
    const ForecastAggregationERC20Proxy = await upgrades.deployProxy(
        ForecastAggregationERC20Object,
        ["ForecastAggregation", "FA", 800_000_000_000, deployer],
        {
            initializer: "initialize",
        }
    );
    // * 等待代理合约部署完成,确保链上已经有合约地址
    await ForecastAggregationERC20Proxy.waitForDeployment();
    // * 获取代理合约地址
    const ForecastAggregationERC20ProxyAddress = await ForecastAggregationERC20Proxy.getAddress();
    // * 获取实现合约(Implementation)的地址
    const ForecastAggregationERC20ImplAddress = await upgrades.erc1967.getImplementationAddress(ForecastAggregationERC20ProxyAddress);
    console.log("ForecastAggregationERC20合约地址：", ForecastAggregationERC20ProxyAddress);
    console.log("ForecastAggregationERC20实现合约地址：", ForecastAggregationERC20ImplAddress);

    const ProtocolAccessUpgradeableProxy = await upgrades.deployProxy(
        ProtocolAccessUpgradeableObject,
        [deployer],
        {
            initializer: "initialize",
        }
    );
    // * 等待代理合约部署完成,确保链上已经有合约地址
    await ProtocolAccessUpgradeableProxy.waitForDeployment();
    // * 获取代理合约地址
    const ProtocolAccessUpgradeableProxyAddress = await ProtocolAccessUpgradeableProxy.getAddress();
    // * 获取实现合约(Implementation)的地址
    const ProtocolAccessUpgradeableImplAddress = await upgrades.erc1967.getImplementationAddress(ProtocolAccessUpgradeableProxyAddress);
    console.log("ProtocolAccessUpgradeable合约地址：", ProtocolAccessUpgradeableProxyAddress);
    console.log("ProtocolAccessUpgradeable实现合约地址：", ProtocolAccessUpgradeableImplAddress);

    const BetRouterUpgradeableProxy = await upgrades.deployProxy(
        BetRouterUpgradeableObject,
        [ProtocolAccessUpgradeableProxyAddress],
        {
            initializer: "initialize",
            kind: "uups"
        }
    );
    await BetRouterUpgradeableProxy.waitForDeployment();
    const BetRouterUpgradeableProxyAddress = await BetRouterUpgradeableProxy.getAddress();
    const BetRouterUpgradeableImplAddress = await upgrades.erc1967.getImplementationAddress(BetRouterUpgradeableProxyAddress);
    console.log("BetRouterUpgradeable合约地址：", BetRouterUpgradeableProxyAddress);
    console.log("BetRouterUpgradeable实现合约地址：", BetRouterUpgradeableImplAddress);

    const FeeVaultProxy = await upgrades.deployProxy(
        FeeVaultObject,
        [ProtocolAccessUpgradeableProxyAddress, ForecastAggregationERC20ProxyAddress],
        {
            initializer: "initialize",
            kind: "uups"
        }
    );
    await FeeVaultProxy.waitForDeployment();
    const FeeVaultProxyAddress = await FeeVaultProxy.getAddress();
    const FeeVaultImplAddress = await upgrades.erc1967.getImplementationAddress(FeeVaultProxyAddress);
    console.log("FeeVault合约地址：", FeeVaultProxyAddress);
    console.log("FeeVault实现合约地址：", FeeVaultImplAddress);

    // const TopicRegistryUpgradeableProxy = await upgrades.deployProxy(
    //     TopicRegistryUpgradeableObject,
    //     [ProtocolAccessUpgradeableProxyAddress],
    //     {
    //         initializer: "initialize",
    //         kind: "uups"
    //     }
    // );
    // await TopicRegistryUpgradeableProxy.waitForDeployment();
    // const TopicRegistryUpgradeableProxyAddress = await TopicRegistryUpgradeableProxy.getAddress();
    // const TopicRegistryUpgradeableImplAddress = await upgrades.erc1967.getImplementationAddress(TopicRegistryUpgradeableProxyAddress);
    // console.log("TopicRegistryUpgradeable合约地址：", TopicRegistryUpgradeableProxyAddress);
    // console.log("TopicRegistryUpgradeable实现合约地址：", TopicRegistryUpgradeableImplAddress);

    // const OracleAdapterUpgradeableProxy = await upgrades.deployProxy(
    //     OracleAdapterUpgradeableObject,
    //     [ProtocolAccessUpgradeableProxyAddress],
    //     {
    //         initializer: "initialize",
    //         kind: "uups"
    //     }
    // );
    // await OracleAdapterUpgradeableProxy.waitForDeployment();
    // const OracleAdapterUpgradeableProxyAddress = await OracleAdapterUpgradeableProxy.getAddress();
    // const OracleAdapterUpgradeableImplAddress = await upgrades.erc1967.getImplementationAddress(OracleAdapterUpgradeableProxyAddress);
    // console.log("OracleAdapterUpgradeable合约地址：", OracleAdapterUpgradeableProxyAddress);
    // console.log("OracleAdapterUpgradeable实现合约地址：", OracleAdapterUpgradeableImplAddress);

    const EscrowVaultUpgradeableProxy = await upgrades.deployProxy(
        EscrowVaultUpgradeableObject,
        [ProtocolAccessUpgradeableProxyAddress, ForecastAggregationERC20ProxyAddress, BetRouterUpgradeableProxyAddress],
        {
            initializer: "initialize",
            kind: "uups"
        }
    );
    await EscrowVaultUpgradeableProxy.waitForDeployment();
    const EscrowVaultUpgradeableProxyAddress = await EscrowVaultUpgradeableProxy.getAddress();
    const EscrowVaultUpgradeableImplAddress = await upgrades.erc1967.getImplementationAddress(EscrowVaultUpgradeableProxyAddress);
    console.log("EscrowVaultUpgradeable合约地址：", EscrowVaultUpgradeableProxyAddress);
    console.log("EscrowVaultUpgradeable实现合约地址：", EscrowVaultUpgradeableImplAddress);

    const SettlementUpgradeableProxy = await upgrades.deployProxy(
        SettlementUpgradeableObject,
        [ProtocolAccessUpgradeableProxyAddress, EscrowVaultUpgradeableProxyAddress, FeeVaultProxyAddress, 100, BetRouterUpgradeableProxyAddress],
        {
            initializer: "initialize",
            kind: "uups"
        }
    );
    await SettlementUpgradeableProxy.waitForDeployment();
    const SettlementUpgradeableProxyAddress = await SettlementUpgradeableProxy.getAddress();
    const SettlementUpgradeableImplAddress = await upgrades.erc1967.getImplementationAddress(SettlementUpgradeableProxyAddress);
    console.log("SettlementUpgradeable合约地址：", SettlementUpgradeableProxyAddress);
    console.log("SettlementUpgradeable实现合约地址：", SettlementUpgradeableImplAddress);

    // ? 2. 进行相关合约的内容记录
    // * 定义本地存储路径,用于保存代理和实现地址及ABI信息
    // 定义存储路径
    const cacheDir = path.join(__dirname, "../.cache");

    // 如果不存在 .cache 目录，则创建
    if (!fs.existsSync(cacheDir)) {
        fs.mkdirSync(cacheDir, { recursive: true });
    }

    const ForecastAggregationERC20StorePath = path.join(__dirname, "../.cache/proxyForecastAggregationERC20.json");
    const ProtocolAccessUpgradeableStorePath = path.join(__dirname, "../.cache/proxyProtocolAccessUpgradeable.json");
    const BetRouterUpgradeableStorePath = path.join(__dirname, "../.cache/proxyBetRouterUpgradeable.json");
    const FeeVaultStorePath = path.join(__dirname, "../.cache/proxyFeeVault.json");
    // const TopicRegistryUpgradeableStorePath = path.join(__dirname, "../.cache/proxyTopicRegistryUpgradeable.json");
    // const OracleAdapterUpgradeableStorePath = path.join(__dirname, "../.cache/proxyOracleAdapterUpgradeable.json");
    const EscrowVaultUpgradeableStorePath = path.join(__dirname, "../.cache/proxyEscrowVaultUpgradeable.json");
    const SettlementUpgradeableStorePath = path.join(__dirname, "../.cache/proxySettlementUpgradeable.json");


    // * 将proxy地址、实现地址、合约ABI写入JSON文件,方便前端或测试脚本使用
    fs.writeFileSync(
        ForecastAggregationERC20StorePath,
        JSON.stringify({
            proxyAddress: ForecastAggregationERC20ProxyAddress,
            implAddress: ForecastAggregationERC20ImplAddress,
            abi: ForecastAggregationERC20Object.interface.format("json"),
        })
    );

    fs.writeFileSync(
        ProtocolAccessUpgradeableStorePath,
        JSON.stringify({
            proxyAddress: ProtocolAccessUpgradeableProxyAddress,
            implAddress: ProtocolAccessUpgradeableImplAddress,
            abi: ProtocolAccessUpgradeableObject.interface.format("json"),
        })
    );

    fs.writeFileSync(
        BetRouterUpgradeableStorePath,
        JSON.stringify({
            proxyAddress: BetRouterUpgradeableProxyAddress,
            implAddress: BetRouterUpgradeableImplAddress,
            abi: BetRouterUpgradeableObject.interface.format("json"),
        })
    );

    fs.writeFileSync(
        FeeVaultStorePath,
        JSON.stringify({
            proxyAddress: FeeVaultProxyAddress,
            implAddress: FeeVaultImplAddress,
            abi: FeeVaultObject.interface.format("json"),
        })
    );

    // fs.writeFileSync(
    //     TopicRegistryUpgradeableStorePath,
    //     JSON.stringify({
    //         proxyAddress: TopicRegistryUpgradeableProxyAddress,
    //         implAddress: TopicRegistryUpgradeableImplAddress,
    //         abi: TopicRegistryUpgradeableObject.interface.format("json"),
    //     })
    // );

    // fs.writeFileSync(
    //     OracleAdapterUpgradeableStorePath,
    //     JSON.stringify({
    //         proxyAddress: OracleAdapterUpgradeableProxyAddress,
    //         implAddress: OracleAdapterUpgradeableImplAddress,
    //         abi: OracleAdapterUpgradeableObject.interface.format("json"),
    //     })
    // );

    fs.writeFileSync(
        EscrowVaultUpgradeableStorePath,
        JSON.stringify({
            proxyAddress: EscrowVaultUpgradeableProxyAddress,
            implAddress: EscrowVaultUpgradeableImplAddress,
            abi: EscrowVaultUpgradeableObject.interface.format("json"),
        })
    );

    fs.writeFileSync(
        SettlementUpgradeableStorePath,
        JSON.stringify({
            proxyAddress: SettlementUpgradeableProxyAddress,
            implAddress: SettlementUpgradeableImplAddress,
            abi: SettlementUpgradeableObject.interface.format("json"),
        })
    );


    // 使用hardhat-deploy保存部署信息
    // save方法会将部署信息写入deployments文件夹中,方便之后使用get方法获取合约实例
    await save("ForecastAggregationERC20Info", {
        address: ForecastAggregationERC20ProxyAddress,
        abi: ForecastAggregationERC20Object.interface.format("json"),
    });
    await save("ProtocolAccessUpgradeableInfo", {
        address: ProtocolAccessUpgradeableProxyAddress,
        abi: ProtocolAccessUpgradeableObject.interface.format("json"),
    });
    await save("BetRouterUpgradeableInfo", {
        address: BetRouterUpgradeableProxyAddress,
        abi: BetRouterUpgradeableObject.interface.format("json"),
    });
    await save("FeeVaultInfo", {
        address: FeeVaultProxyAddress,
        abi: FeeVaultObject.interface.format("json"),
    });
    // await save("TopicRegistryUpgradeableInfo", {
    //     address: TopicRegistryUpgradeableProxyAddress,
    //     abi: TopicRegistryUpgradeableObject.interface.format("json"),
    // });
    // await save("OracleAdapterUpgradeableInfo", {
    //     address: OracleAdapterUpgradeableProxyAddress,
    //     abi: OracleAdapterUpgradeableObject.interface.format("json"),
    // });
    await save("EscrowVaultUpgradeableInfo", {
        address: EscrowVaultUpgradeableProxyAddress,
        abi: EscrowVaultUpgradeableObject.interface.format("json"),
    });
    await save("SettlementUpgradeableInfo", {
        address: SettlementUpgradeableProxyAddress,
        abi: SettlementUpgradeableObject.interface.format("json"),
    });

};

// ? 3. 打上对应的部署/升级标签
// 为hardhat-deploy定义标签,用于选择性部署
module.exports.tags = ["ForecastAggregation_V1"]; 