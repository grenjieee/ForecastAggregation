const { ethers, deployments, upgrades, network } = require("hardhat");
const { expect } = require("chai");

describe("Test", function () {
    let deployer;
    let EscrowVaultData, tokenData, BetRouterData;
    let EscrowVault, Token, BetRouter;

    before(async function () {
        this.timeout(600000);
        [deployer] = await ethers.getSigners();
        console.log("当前网络:", network.name);
        console.log("Deployer地址:", deployer.address);

        const fs = require("fs");
        const path = require("path");

        // 获取已部署的合约信息
        EscrowVaultData = JSON.parse(fs.readFileSync(path.join(__dirname, "../.cache/proxyEscrowVaultUpgradeable.json")));
        tokenData = JSON.parse(fs.readFileSync(path.join(__dirname, "../.cache/proxyForecastAggregationERC20.json")));
        BetRouterData = JSON.parse(fs.readFileSync(path.join(__dirname, "../.cache/proxyBetRouterUpgradeable.json")));

        EscrowVault = await ethers.getContractAt(EscrowVaultData.abi, EscrowVaultData.proxyAddress);
        Token = await ethers.getContractAt(tokenData.abi, tokenData.proxyAddress);
        BetRouter = await ethers.getContractAt(BetRouterData.abi, BetRouterData.proxyAddress);

    });

    it("test", async function () {
        this.timeout(300000);

        console.log("\n====== Step2 检查 betTimestamp ======");
        const betId = "0x1abf92fbf8b3c240dbbca56b6c4358be16dc5ce0cd6e4601e61e9154252a9964";
        const wallet = deployer;
        const decimals = await Token.decimals();
        const amount = ethers.parseUnits("0.49", decimals);

        const betTimestamp = await BetRouter.getBetTimestamp(betId);
        console.log("betTimestamp:", betTimestamp.toString());

        const now = Math.floor(Date.now() / 1000);
        console.log("now:", now);

        const lockDeadline = Number(betTimestamp) + 3600;

        console.log("lockDeadline:", lockDeadline);

        if (now > lockDeadline) {
            console.log("❌ 已超出锁定时间");
        } else {
            console.log("✅ 未超时");
        }

        console.log("\n====== Step3 检查 Token Balance ======");
        const balance = await Token.balanceOf(wallet.address);
        console.log("balance:", balance.toString());

        console.log("\n====== Step4 检查 Allowance ======");
        console.log("EscrowVault地址:", EscrowVaultData.proxyAddress);
        const allowance = await Token.allowance(wallet.address, EscrowVaultData.proxyAddress);
        console.log("allowance:", allowance.toString());

        if (allowance < amount) {
            console.log("❌ allowance 不足");
        } else {
            console.log("✅ allowance 足够");
        }

        console.log("\n====== Step5 staticCall 模拟执行 ======");

        try {

            await EscrowVault.executedFunds.staticCall(betId, "0x1111111111111111111111111111111111111111111111111111111111111234", "0x860ba080a8074246C26D53ed1a7f9A3D21850098", amount);

            console.log("✅ staticCall 成功");

        } catch (err) {

            console.log("❌ staticCall revert");

            if (err.data) {
            console.log("revert data:", err.data);
            }

            if (err.reason) {
            console.log("revert reason:", err.reason);
            }

            console.log(err);
        }
    });


});


