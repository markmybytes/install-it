<a id="readme-top"></a>


<!-- PROJECT SHIELDS -->
<div align="center">

  [![Tag][tag-shield]][tag-url]
  [![Contributors][contributors-shield]][contributors-url]
  [![Forks][forks-shield]][forks-url]
  [![Stargazers][stars-shield]][stars-url]
  [![Issues][issues-shield]][issues-url]
  [![MIT License][license-shield]][license-url]

</div>


<!-- PROJECT LOGO -->
<div align="center">
  <a href="https://github.com/markmybytes/install-it">
    <img src="https://github.com/user-attachments/assets/ea47a738-6f1e-4e8d-bde0-4f12118ff103" alt="Logo" width="80" height="80">
  </a>

  <h3 align="center">install-it</h3>

  <p align="center">
    A Driver/Software Installation Tool
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
## About The Project

<p align="center">
  <img src="https://github.com/user-attachments/assets/35606055-7ce6-4e97-8152-a7042d7fe001" width="754" height="569">
</p>

install-it is a PC setup assistant tool that aims to simplify and speed up the driver installation process. <br />
It allows you to **preload a bunch of driver installers** and then **select the most suitable ones to install** during the setup of a new PC. <br />
Beyond drivers, installing softwares, and executing commands are also possible in install-it, see [Usage](#usage) section for more.

| Download :arrow_down: | [Latest Release](https://github.com/markmybytes/install-it/releases/latest) |
|-----------------------|-----------------------------------------------------------------------------|

> [!NOTE]  
> install-it does not include any driver installers in releases. You may check out [it-claws](https://github.com/markmybytes/it-claws), a CLI tool that automatically download common hardware drivers.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

[<img src="https://img.shields.io/badge/font%20awesome-538cd7?style=for-the-badge&logo=fontawesome&logoColor=white">](https://fontawesome.com/)
[<img src="https://img.shields.io/badge/go-01add8?style=for-the-badge&logo=go&logoColor=white">](https://go.dev/)
[<img src="https://img.shields.io/badge/tailwindcss-38bdf8?style=for-the-badge&logo=tailwindcss&logoColor=white">](https://tailwindcss.com/)
[<img src="https://img.shields.io/badge/vue.js-41b883?style=for-the-badge&logo=vue.js&logoColor=white">](https://vuejs.org/)
[<img src="https://img.shields.io/badge/wails-d32a2d?style=for-the-badge&logo=wails&logoColor=white">](https://wails.io/)

<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- GETTING STARTED -->
## Getting Started

### Prerequisites

- Go ≥ 1.23 https://go.dev/doc/install
- Node 22 https://nodejs.org/en/download/package-manager

### Setup

#### Install dependencies

- Wails
  ```sh
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

- NPM Dependencies
  ```sh
  cd ./frontend
  npm install
  ```

#### Commands

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

<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- USAGE EXAMPLES -->
## Usage

### Managing Installers

<img src="https://github.com/user-attachments/assets/909dcbbd-9b02-4c06-941e-a77035e1250f" width="754" height="569">

To add an installer into install-it:

1. Place all your installer under the `driver/<category>` folder
2. Create a installer group
   - you can add multiple installer into a single group
   - install-it predefined three categroies: `network`, `display`, and `miscellaneous`, only `miscellaneous` allow multiple selection
3. Enter the details for each installer
4. Done

> [!TIP]
> It is highly recommended to provide the correct command-line options so that the installer can be executed in unattended mode to maximise functionality of install-it. <br />
> See [Exection Option](#exection-option) section for more information.

<details>
  <summary>[Example] Execute commands</summary>

  You can execute binary available in the OS `PATH` variable. In Windows, you can use CMD or Powershell to execute commands or scripts.

  For CMD, you can execute commands by entering `cmd` in the path field, and `/c,<command>` in the option field. Then it is equivlent to:

  ```batch
  cmd /c command
  ```

  For Powershell, you can execute commands by entering `powershell` in the path field, and `-Command,<command>` in the option field. Then it is equivlent to:

  ```batch
  powershell -Command command
  ```
</details>

<details>
  <summary>[Example] Install non-driver software</summary>

  Software installer usually provides a slient install options like driver installers. For example, Steam support silent install by supplying `/S` option when you executing `SteamSetup.exe`. Explore yourself and turn install-it to your PC setup toolbox :)
</details>

### Execution Option

[Command-line Option/Argument](https://en.wikipedia.org/wiki/Command-line_interface#Arguments) is to control the execution behaviour, or input data into the program. <br />
Many installers support unattend mode or silent mode, where the software will be installed automatically without any interactions.

install-it provides installation parameters preset for the following common drivers:

| Option         | Applicable Installer                                                                                                                                             |
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

For software that is not in the preset, you can try searching online with `software name` + `silent`/`unattended`/`command line install`.

### Installation

Select all the suitable software in the home page and click `Execute`. A popup will be displayed for execution status.
 
> [!IMPORTANT]  
> install-it uses the exit status code to determine the execution status. Some programs may return 0 (indicating successful) even if the installation not yet completed, or failed.

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
[license-url]: https://github.com/markmybytes/install-it/blob/master/LICENSE
