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

<h3 align="center">Go-Chronos (provisional)</h3>

  <p align="center">
    A PoVF-base blockchain node implemented with Go-lang
    <br />
    <br />
  </p>
</div>


<!-- ABOUT THE PROJECT -->
## About The Project

Chronos is a blockchain node based on Proof of Verifiable Function (PoVF). Based on the unpredictability of verifiable 
functions, PoVF provides a fair and decentralized consensus mechanism.

<!-- GETTING STARTED -->
## Getting Started

### Build & Run node

1. Clone the repo
   ```sh
   git clone https://github.com/Chain-Lab/go-chronos.git
   ```
2. Build node
   ```sh
   cd ./cmd/chronos
   go build
   ```
3.1. Run a genesis chronos node
   ```sh
   ./chronos -d [data_path] -g -c [config_path]
   ```
3.2. Run general node
   ```sh
   ./chronos -d [data_path] -c [config_path] -b [bootstrap_url]
   ```


<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE.txt` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/Chain-Lab/go-chronos.svg?style=for-the-badge
[contributors-url]: https://github.com/Chain-Lab/go-chronos/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/Chain-Lab/go-chronos.svg?style=for-the-badge
[forks-url]: https://github.com/Chain-Lab/go-chronos/network/members
[stars-shield]: https://img.shields.io/github/stars/Chain-Lab/go-chronos.svg?style=for-the-badge
[stars-url]: https://github.com/Chain-Lab/go-chronos/stargazers
[issues-shield]: https://img.shields.io/github/issues/Chain-Lab/go-chronos.svg?style=for-the-badge
[issues-url]: https://github.com/Chain-Lab/go-chronos/issues
[license-shield]: https://img.shields.io/github/license/Chain-Lab/go-chronos.svg?style=for-the-badge
[license-url]: https://github.com/Chain-Lab/go-chronos/blob/master/LICENSE.txt