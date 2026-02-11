require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();
require("hardhat-deploy");
require("@openzeppelin/hardhat-upgrades");
require("@nomicfoundation/hardhat-foundry");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: {
    version: "^0.8.0",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200,
      },
    },
  },
  paths: {
    sources: "./contracts",
    artifacts: "./out", // 指向 Foundry 输出目录
  },
  namedAccounts: {
    deployer: 0,
    user1: 1,
    user2: 2,
    user3: 3,
    user4: 4,
  },
  networks: {
    sepolia: {
      url: `https://sepolia.infura.io/v3/${process.env.INFURA_API_KEY}`,
      accounts: [
        process.env.METAMASK1_PRIVATE_KEY,
      ],
    },
  },
};
