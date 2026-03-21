<div align="center">

  <img width="570" height="86" alt="PixPin_2026-03-19_17-22-10" src="https://github.com/user-attachments/assets/024017d8-14db-4b80-af4d-a2c3024f2d69" />

# SpireModGo

`SpireModGo` 是一个面向 **杀戮尖塔2** 的mod和存档管理工具，提供基于go的TUI页面。
</div>

## 下载安装

从[release页](https://github.com/LingyeNBird/SpireModGo/releases)选择最新版本对应平台下载即可。
> 如果网络不畅，可以从[cnb的镜像仓库](https://cnb.cool/lingyeSelf/SpireModGo/-/releases)下载



## 使用说明

### 模组管理

#### 1. 导入与安装模组

在模组管理页面点击导入即可
<img width="1730" height="1044" alt="image" src="https://github.com/user-attachments/assets/82d53959-967c-4e95-a2a3-55f0b480c886" />



导入到未安装，是放置到本地，可以后续点击**安装**来安装
导入到已安装，是直接放到游戏模组文件夹里

#### 2. 卸载模组

卸载模组会直接从游戏模组文件夹中删除该模组
<img width="1730" height="1044" alt="image" src="https://github.com/user-attachments/assets/64e5d64a-e3e5-4ddc-baac-069946627645" />



#### 3. 导出模组
会将选中的模组打包成一个zip压缩包，可以被本工具导入
方便与朋友同步模组，解决有些mod需要双方一致才能开房间的问题

**安装模组是，会提醒是否复制原版存档至模组存档，这是因为杀戮尖塔的原版存档和模组存档相互独立**

**本工具提供一键复制功能，可以将原版存档复制到模组存档。相同的，如果模组不想玩了想玩原版，也可用本工具反向复制**

#### 4. 模组修复
最近因为杀戮尖塔的更新修改的模组的格式，导致存在旧模组打不开的问题，解决方法可以参考b站up：[千层雪_Yuki](https://www.bilibili.com/video/BV1uywaz8Ei6)

本工具也提供了一键修复方式
<img width="1730" height="1044" alt="image" src="https://github.com/user-attachments/assets/54ed424f-7d6f-4182-a3bd-6f3a8e63e67b" />



### 存档管理
本工具会扫描所有steam用户，不同用户的存档是独立的，可以切换不同的用户
<img width="1730" height="1044" alt="image" src="https://github.com/user-attachments/assets/0fea797d-d039-4f0f-af28-ddd284cc10cd" />


点击复制会将当前选中的存档复制到选定的复制槽位

本工具还提供备份相关功能

### 设置
程序进入后会自动扫描游戏安装路径

在设置页可以手动设置游戏路径，避免识别不了
<img width="865" height="522" alt="PixPin_2026-03-20_09-40-58" src="https://github.com/user-attachments/assets/af515343-1cc4-4118-b3f2-3f7f9b790bf8" />

设置页还可以检查更新

### 运行要求
- 已安装并至少启动过一次 **Slay the Spire 2**，这样 Steam 存档目录才会生成
- 游戏目录中存在 `SlayTheSpire2.exe`

## 开发者
### 本地构建
Go 1.25.0 或兼容版本，见 `go.mod`

在仓库根目录执行：

```bash
go build -o dist/SpireModGo.exe .
```

如果你想和发布工作流保持一致，可使用同样的环境变量思路：

```bash
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags "-s -w" -o dist/SpireModGo.exe .
```

### 运行

```bash
dist\SpireModGo.exe
```

### 仓库结构

```text
.
├─ .github/workflows/release.yml   # Windows 手动发布流程
├─ internal/manager/               # 配置、模组、存档、Steam 路径逻辑
├─ internal/ui/                    # TUI 页面与交互
├─ Mods/                           # 随仓库分发的本地模组源目录
├─ dist/                           # 构建输出目录
└─ main.go                         # 程序入口
```

运行时生成的配置和日志会写入用户目录：

```text
%APPDATA%\SpireModGo\modmanager.json
%APPDATA%\SpireModGo\logs
```

除了仓库内的 `Mods/` 目录，程序还会读取用户目录下的本地模组源：

```text
%APPDATA%\SpireModGo\mods
```

界面中显示的未安装模组，实际可能是以下两者之一，或同时存在：

- 仓库根目录下的 `Mods/`
- `%APPDATA%\SpireModGo\mods`

---
> ## 致谢
> 本工具基本上是从b站up [皮一下就很凡](https://space.bilibili.com/26786884) 的脚本改动而来
> 
> 哪怕不给本仓库star，也请给他点个关注。
> 
> 同时，再此感谢所有杀戮尖塔模组作者，你们开发的模组很好用。
