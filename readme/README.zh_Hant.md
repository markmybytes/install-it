<a id="readme-top"></a>

<!-- PROJECT SHIELDS -->
<div align="center">
  
  [![Tag][tag-shield]][tag-url]
  [![Contributors][contributors-shield]][contributors-url]
  [![Forks][forks-shield]][forks-url]s
  [![Stargazers][stars-shield]][stars-url]
  [![Issues][issues-shield]][issues-url]
  [![MIT License][license-shield]][license-url]
  
</div>
Ks

<!-- PROJECT LOGO -->
<div align="center">
  <a href="https://github.com/markmybytes/install-it">
    <img src="https://github.com/user-attachments/assets/ea47a738-6f1e-4e8d-bde0-4f12118ff103" alt="Logo" width="80" height="80">
  </a>

  <h3 align="center">install-it</h3>

  <p align="center">
    程式／軀動安裝工具
    <br>
    <a href="https://github.com/markmybytes/install-it/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
    ·
    <a href="https://github.com/markmybytes/install-it/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
  </p>

  <p align="center">
    <a href="https://github.com/markmybytes/install-it//README.md">English</a>
    ·
    <a href="https://github.com/markmybytes/install-it/readme/README.zh_Hant.md">繁體中文</a>
  </p>
</div>


<!-- ABOUT THE PROJECT -->
## Project 簡介

<p align="center">
  <img src="https://github.com/user-attachments/assets/a8c6f316-87d7-4b9f-8ccd-6a732326643d" width="754" height="569">
<p align="right">

install-it 是一個軀動程式安裝工具，旨在減少安裝大量硬件軀動所需的時間。它允許您預先加入驅動程式安裝檔至工具內，並在新電腦設定過程中選擇最合適的驅動程式進行安裝。<br />
除了驅動程式之外，install-it 還支援安裝軟體及執行指令，詳情請參閱[使用](#使用)。

| 下載 :arrow_down: | [Latest Release](https://github.com/markmybytes/install-it/releases/latest) |
|-------------------|-----------------------------------------------------------------------------|

> [!NOTE]
> install-it 並沒有內置任何軀動程式。如有需要，可使用 [driver-claw](https://github.com/markmybytes/driver-claw) 工具，一鍵自動下載所有最新的常見硬件軀動程式。


<p align="right">(<a href="#readme-top">回到最頂</a>)</p>

### 第三方工具使用

[<img src="https://img.shields.io/badge/font%20awesome-538cd7?style=for-the-badge&logo=fontawesome&logoColor=white">](https://fontawesome.com/)
[<img src="https://img.shields.io/badge/go-01add8?style=for-the-badge&logo=go&logoColor=white">](https://go.dev/)
[<img src="https://img.shields.io/badge/tailwindcss-38bdf8?style=for-the-badge&logo=tailwindcss&logoColor=white">](https://tailwindcss.com/)
[<img src="https://img.shields.io/badge/vue.js-41b883?style=for-the-badge&logo=vue.js&logoColor=white">](https://vuejs.org/)
[<img src="https://img.shields.io/badge/wails-d32a2d?style=for-the-badge&logo=wails&logoColor=white">](https://wails.io/)

<p align="right">(<a href="#readme-top">回到最頂</a>)</p>


<!-- GETTING STARTED -->
## 開發

### 所需軟件

- Go ≥ 1.23 https://go.dev/doc/install
- Node 22 https://nodejs.org/en/download/package-manager

### 安裝 Dependency

- Wails
  ```sh
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```
- NPM Dependencies
  ```sh
  cd ./frontend
  npm install
  ```

### 常用指令

- Debug run

  ```sh
  wails dev
  ```

- Build Executable
  ```sh
  wails build

  # build with version number set
  wails build -ldflags "-X main.buildVersion=<version number>"
  ```

<p align="right">(<a href="#readme-top">回到最頂</a>)</p>


<!-- USAGE EXAMPLES -->
## 使用

### 安裝程式管理

<img src="https://github.com/user-attachments/assets/8fb85b19-133e-4cbf-9ee4-21e5237c9089" width="754" height="569">

加入安裝程式至 install-it 內的步驟：

1. 裝安裝程式的檔案移動至 `driver/<category>` 資料夾內
2. 新增一個「安裝程式群組」
     - 你可以加入多個安裝程式至同一個群組內
     - install-it 提供了三個分類可供選擇：`網絡介面卡`、`顯示卡`及`其他`，只有`其他`可以多選
3. 輸入安裝程式的詳細資料
4. 完成

> [!TIP]
> 為更好發揮 install-it 的功能，請務必為所有安裝程式輸入相應的命令列選項（Command-line option），使其能以零人手操作的模式（unattended）下自動執行。

<details>
  <summary>[範例] 執行指令</summary>

  install-it 支援執行在作業系統 `PATH` 的環境變數內的程序。在 Windows 上，你可以透過 CMD 或 Powershell 執行指令碼。

  就 CMD 而言，輸入 `cmd` 至路徑中；`/c,<command>` 至執行參數中，即可執行 CMD 指令：

  ```batch
  cmd /c command
  ```

  就 Powershell 而言，輸入 `powershell` 至路徑中；`-Command,<command>` 至執行參數中，即可執行 powershell 指令：

  ```batch
  powershell -Command command
  ```
</details>

<details>
  <summary>[範例] 安裝非軀動程式的軟件</summary>

  一般而言，軟件安裝程式會提供零人手操作安裝的模式。例如 Steam 可以加入 `/S` 的參數執行 `SteamSetup.exe`，即可在零人手操作下進行安裝。
</details>

### 執行參數

[程式命令行選項](https://en.wikipedia.org/wiki/Command-line_interface#Arguments) 是用作控制程式的行為或輸入資料至程式中。大部份安裝程式都設有「零人手操作模式」（unattend/silent mode），即整個安裝過程中，毋須進行進何操作。

而 install-it 內置了一些常見軀動程式的零人手操作模式參數：

| 選項           | 適用的軀動程式                                                                                                                                                    |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Intel LAN      | [Intel® Ethernet Adapter Complete Driver Pack](https://www.intel.com/content/www/us/en/download/15084/intel-ethernet-adapter-complete-driver-pack.html)          |
| Realtek LAN    | [Realtek PCIe FE / GBE / 2.5G / 5G Ethernet Family Controller Software](https://www.realtek.com/Download/List?cate_id=584)                                       |
| Nvidia Display | [GeForce Game Ready Driver/Nvidia Studio Driver](https://www.nvidia.com/en-us/drivers/)                                                                          |
| AMD Display    | [AMD Software: Adrenalin Edition](https://www.amd.com/en/support/download/drivers.html)                                                                          |
| Intel Display  | [Intel® Arc™ & Iris® Xe Graphics/7th-10th Gen Processor Graphics](https://www.intel.com/content/www/us/en/support/articles/000090440/graphics.html)              |
| Intel WiFi     | [Intel® Wireless Wi-Fi Drivers](https://www.intel.com/content/www/us/en/download/19351/intel-wireless-wi-fi-drivers-for-windows-10-and-windows-11.html)          |
| Intel BT       | [Intel® Wireless Bluetooth® Drivers](https://www.intel.com/content/www/us/en/download/18649/intel-wireless-bluetooth-drivers-for-windows-10-and-windows-11.html) |
| Intel Chipset  | [Chipset INF Utility](https://www.intel.com/content/www/us/en/support/products/1145/software/chipset-software/intel-chipset-software-installation-utility.html)  |
| AMD Chipset    | [AMD Chipset Drivers](https://www.amd.com/en/support/download/drivers.html)                                                                                      |

對於未有在上述表格中例出的軟件，你可嘗試以 `software name` + `silent`/`unattended`/`command line install` 搜索。

### 安裝


在首頁選擇所有合適的軟體後，點擊「執行」即可開始安裝。系統會跳出提示視窗，顯示執行狀態。
 
> [!IMPORTANT]  
> install-it 會根據程式的退出狀態碼來判斷執行結果。但有些程式即使安裝尚未完成或失敗，仍可能回傳 0（代表成功），因此執行狀態可能不完全準確。

<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- MARKDOWN LINKS & IMAGES -->
[tag-url]: https://github.com/markmybytes/install-it/releases
[tag-shield]: https://img.shields.io/github/v/tag/markmybytes/install-it?style=for-the-badge&label=LATEST&color=%23B1B1B1
[contributors-shield]: https://img.shields.io/github/contributors/markmybytes/install-it.svg?style=for-the-badge
[contributors-url]: https://github.com/markmybytes/install-it/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/markmybytes/install-it.svg?style=for-the-badge
[forks-url]: https://github.com/markmybytes/install-it/network/members
[stars-shield]: https://img.shields.io/github/stars/markmybytes/install-it.svg?style=for-the-badge
[stars-url]: https://github.com/markmybytes/install-it/stargazers
[issues-shield]: https://img.shields.io/github/issues/markmybytes/install-it.svg?style=for-the-badge
[issues-url]: https://github.com/markmybytes/install-it/issues
[license-shield]: https://img.shields.io/github/license/markmybytes/install-it.svg?style=for-the-badge
[license-url]: https://github.com/markmybytes/install-it/blob/master/LICENSE.txt
