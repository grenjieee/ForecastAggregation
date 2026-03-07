const enforceUseClientForFeatures = require("./eslint-rules/enforce-use-client-for-features.js");

module.exports = {
  overrides: [
    {
      files: ["**/*.tsx", "**/*.ts"], // 仅检查 TypeScript 和 TSX 文件
      excludedFiles: ["**/components/**", "**/hooks/**"], // 排除客户端组件目录
      rules: {
        "custom/enforce-use-client-for-features": "error", // 启用自定义规则
      },
    },
  ],
};