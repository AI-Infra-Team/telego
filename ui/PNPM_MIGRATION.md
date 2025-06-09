# Telego UI - pnpm 迁移指南

## 为什么选择 pnpm？

pnpm (Performant npm) 是一个快速、节省磁盘空间的包管理器，具有以下优势：

### 🚀 性能优势
- **安装速度快**：并行安装，比npm快2-3倍
- **节省磁盘空间**：通过硬链接和符号链接，可节省70%以上磁盘空间
- **严格的依赖管理**：避免幽灵依赖问题

### 📦 兼容性
- **完全兼容npm**：可以直接替换npm命令
- **支持所有npm特性**：包括scripts、workspaces等
- **更好的Monorepo支持**：原生支持工作空间

## 安装 pnpm

### 全局安装
```bash
# 使用npm安装 (推荐)
npm install -g pnpm

# 或使用官方安装脚本
curl -fsSL https://get.pnpm.io/install.sh | sh -

# 或使用homebrew (macOS)
brew install pnpm
```

### 验证安装
```bash
pnpm --version
```

## 迁移步骤

### 0. 环境准备 (如果遇到问题)
如果在安装过程中遇到环境问题，请先运行项目根目录的环境设置脚本：
```bash
# 回到项目根目录
cd ..

# 运行环境设置脚本
python3 0.dev_env_setup.py

# 回到ui目录
cd ui/
```

### 1. 删除旧的npm依赖文件
```bash
# 在ui目录下执行
rm -rf node_modules/
rm -f package-lock.json
```

### 2. 使用pnpm安装依赖
```bash
pnpm install
```

### 3. 验证迁移结果
```bash
# 检查是否生成了pnpm-lock.yaml
ls -la pnpm-lock.yaml

# 测试构建
pnpm run build

# 测试开发服务器
pnpm run dev
```

## 命令对照表

| npm 命令 | pnpm 命令 | 说明 |
|---------|----------|------|
| `npm install` | `pnpm install` | 安装依赖 |
| `npm install pkg` | `pnpm add pkg` | 添加依赖 |
| `npm install -D pkg` | `pnpm add -D pkg` | 添加开发依赖 |
| `npm uninstall pkg` | `pnpm remove pkg` | 移除依赖 |
| `npm run script` | `pnpm run script` | 运行脚本 |
| `npm run script` | `pnpm script` | 运行脚本(简写) |
| `npm ci` | `pnpm install --frozen-lockfile` | CI环境安装 |

## 项目构建

### 使用Python构建脚本 (推荐)
```bash
# 开发模式
python3 build.py --dev

# 生产构建
python3 build.py

# 构建并预览
python3 build.py --preview

# 清理
python3 build.py --clean
```

### 使用Bash脚本
```bash
./build.sh
```

### 直接使用pnpm命令
```bash
# 安装依赖
pnpm install

# 开发服务器
pnpm run dev

# 生产构建
pnpm run build

# 预览构建结果
pnpm run preview
```

## 配置说明

### .npmrc 配置
项目已配置了合理的pnpm默认设置：
- 自动安装peer依赖
- 显示安装进度
- CI环境自动使用frozen-lockfile

### package.json 更改
- 添加了 `packageManager` 字段指定pnpm版本
- 添加了 `engines` 字段限制Node.js和pnpm版本

## 常见问题

### Q: 如何在CI/CD中使用pnpm？
A: 在CI脚本中使用 `pnpm install --frozen-lockfile` 确保版本一致性。

### Q: 如何处理peer依赖警告？
A: 项目已配置 `auto-install-peers=true`，会自动安装peer依赖。

### Q: 如何使用国内镜像？
A: 在 `.npmrc` 中取消注释 `registry` 配置行。

### Q: 遇到符号链接问题怎么办？
A: 在 `.npmrc` 中启用 `symlink=true` 配置。

### Q: 遇到环境问题怎么办？
A: 先运行项目根目录的环境设置脚本：`cd .. && python3 0.dev_env_setup.py`

### Q: 遇到TypeScript类型定义错误怎么办？
A: 如果遇到 "Cannot find type definition file for 'node'" 类似错误，说明缺少类型定义包：
```bash
# 添加Node.js类型定义
pnpm add -D @types/node

# 如果需要其他类型定义
pnpm add -D @types/其他包名
```

## 回滚方案

如果需要回滚到npm：
```bash
# 删除pnpm文件
rm -rf node_modules/ pnpm-lock.yaml

# 使用npm重新安装
npm install
```

## 更多资源

- [pnpm官方文档](https://pnpm.io/)
- [pnpm CLI文档](https://pnpm.io/cli/add)
- [从npm迁移到pnpm](https://pnpm.io/motivation)