<!-- PROJECT SHIELDS -->
<!--
*** I'm using markdown "reference style" links for readability.
*** Reference links are enclosed in brackets [ ] instead of parentheses ( ).
*** See the bottom of this document for the declaration of the reference variables
*** for contributors-url, forks-url, etc. This is an optional, concise syntax you may use.
*** https://www.markdownguide.org/basic-syntax/#reference-style-links
-->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]

<!-- PROJECT LOGO -->
<br />
<div align="center">

<h3 align="center">Go-Norn</h3>

  <p align="center">
    一个通过 Go-lang 实现的基于可验证随机函数的区块链节点
    <br />
    <br />
  </p>
</div>


<!-- ABOUT THE PROJECT -->
## 关于项目

Norn 是一个基于可验证函数的区块链节点。基于可验证函数的随机性，PoVF 提供了一种公平、去中心化的共识机制。

<!-- GETTING STARTED -->
## 开始

### 编译&&运行节点

1. 克隆仓库
   ```sh
   git clone https://github.com/Chain-Lab/go-norn.git
   ```
2. 编译节点
   ```sh
   cd ./cmd/norn
   go build
   ```

3. 生成配置文件
   ```sh
   cd ./cmd/generate
   go build
   ./generate
   ```

4. 运行一个创世节点，`data_path` 指定 levelDB 的数据存储目录，`config_path` 指定配置文件路径
   ```sh
   ./norn -d [data_path] -g -c [config_path]
   ```
5. 运行一个普通节点，`bootstrap_url` 指定启动节点路径
   ```sh
   ./norn -d [data_path] -c [config_path] -b [bootstrap_url]
   ```


<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE.txt` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/Chain-Lab/go-norn.svg?style=for-the-badge
[contributors-url]: https://github.com/Chain-Lab/go-norn/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/Chain-Lab/go-norn.svg?style=for-the-badge
[forks-url]: https://github.com/Chain-Lab/go-norn/network/members
[stars-shield]: https://img.shields.io/github/stars/Chain-Lab/go-norn.svg?style=for-the-badge
[stars-url]: https://github.com/Chain-Lab/go-norn/stargazers
[issues-shield]: https://img.shields.io/github/issues/Chain-Lab/go-norn.svg?style=for-the-badge
[issues-url]: https://github.com/Chain-Lab/go-norn/issues
[license-shield]: https://img.shields.io/github/license/Chain-Lab/go-norn.svg?style=for-the-badge
[license-url]: https://github.com/Chain-Lab/go-norn/blob/master/LICENSE.txt