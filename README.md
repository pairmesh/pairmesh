![](docs/images/logo.svg#gh-light-mode-only)
![](docs/images/logo_dark.svg#gh-dark-mode-only)


[![LICENSE](https://img.shields.io/github/license/pairmesh/pairmesh.svg)](https://github.com/pairmesh/pairmesh/blob/master/LICENSE)
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Build Status](https://github.com/pairmesh/pairmesh/actions/workflows/build.yml/badge.svg)](https://github.com/pairmesh/pairmesh/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/pairmesh/pairmesh)](https://goreportcard.com/report/github.com/pairmesh/pairmesh)
[![GitHub release](https://img.shields.io/github/tag/pairmesh/pairmesh.svg?label=release)](https://github.com/pairmesh/pairmesh/releases)
[![GitHub release date](https://img.shields.io/github/release-date/pairmesh/pairmesh)](https://github.com/pairmesh/pairmesh/releases)
[![Coverage Status](https://codecov.io/gh/pairmesh/pairmesh/branch/master/graph/badge.svg)](https://codecov.io/gh/pairmesh/pairmesh)
[![GoDoc](https://img.shields.io/badge/Godoc-reference-blue.svg)](https://godoc.org/github.com/pairmesh/pairmesh)

## What is PairMesh?

__The mission of PairMesh is to provide network infrastructure for remote collaboration and work__.

By setting up a security P2P virtual private LAN network among multiple devices to solve the networking problems of off-site network access and remote collaboration. It can easily meet the requirments of remote work collaboration, remote debugging, intranet penetration, NAS/Git server access, etc.

* **Easy to use**

    PairMesh node discovery peers throgh __PairMesh Control Plane__ and configuration of virtual network devices to set up a virtual network in seconds automatically, avoiding any manual configuration steps. The client provides a UI interface to display device information, network topology and node operations. PairMesh Control Plane provides a web dashboard for PairMesh node operations, key management and network maintainence.

* **Blazing Fast**

    PairMesh automatically forms P2P networks with other PairMesh nodes after obtaining network topology information from the PairMesh Control Plane. The network traffic will bypass a central service and transmit to peer nodes directly, making communication between networks formed by PairMesh extremely fast.

* **Security**

    Using the latest security framework [Noise Protocol](https://noiseprotocol.org/noise.html) and the latest cryptography technologies, multiple security mechanisms are included in the communication between Pairmesh and Control Plane as well as between PairMesh nodes to ensure data integrity, security, and speed of traffic between PairMesh networks while having the best encryption and decryption performance.

    - All requests from PairMesh nodes and Control Plane need to verify the Token bound with machine information to ensure that illegal nodes can be prevented from joining the network even if the Token is leaked.
    - The TCP connections between PairMesh and Relay servers will be verified with the signature of node information in the handshake phase, and only those nodes whose information is signed by Control Plane can be used for subsequent communication.
    - All nodes use different keys, and each node pair of the whole topology network uses a unique key to avoid key leakage affecting the security of the whole network.

## Quick Start
- Prerequisites
    - Golang >= v1.17.
- How to build PairMesh
    - Linux/MacOS Terminal
      ```shell
      make pairmesh
      ```
    - Windows Powershell
      ```
      go build -o bin/PairMesh.exe  -ldflags "-s -w -H=windowsgui -X github.com/pairmesh/pairmesh/version.GitHash=$(git describe --no-match --always --dirty) -X github.com/pairmesh/pairmesh/version.GitBranch=$(git rev-parse --abbrev-ref HEAD)" ./cmd/pairmesh
      ```
- Use **[PairMesh](https://www.pairmesh.com)** managed control plane service (Chinese Only).
- Self-hosted PairMesh control plane.

## Architecture

The following diagram shows the overall architecture of PairMesh, where the PairMesh node is an application installed on the end device, responsible for managing the local virtual NIC device and discovering information about other nodes from the Control Plane service, as well as establishing P2P communication connections with other nodes, encrypting and decrypting data and processing traffic data in the network.

![Architecture](./docs/images/architecture.svg#gh-light-mode-only)
![Architecture](./docs/images/architecture_dark.svg#gh-dark-mode-only)

## Community

You can join these discussion forum and chats to discuss and ask PairMesh related questions:

- [Slack channel](https://pairmesh.slack.com)

## Contributing

PairMesh is developed by an open and friendly community. Everybody is cordially welcome to join the community and contribute to PairMesh. We value all forms of contributions, including, but not limited to:

- Code reviewing of the existing patches
- Documentation and usage examples
- Community participation in forums and issues
- Code readability and developer guide
- We welcome contributions that add code comments and code refactor to improve readability
- We also welcome contributions to docs to explain the design choices of the internal
- Test cases to make the codebase more robust
- Tutorials, blog posts, talks that promote the project

Here are guidelines for contributing to various aspect of the project:

- [How to contribute](./CONTRIBUTING.md)
- [PairMesh GitHub workflow](./docs/guide/github-workflow.md)

Any other question? Reach out to the [PairMesh Discussion](https://github.com/pairmesh/pairmesh/discussions) forum to get help!

## Development Guidelines

### Version
Each time when initiating a releasse, there are places to notice to bump up verions:
* `version/version.go: MajorVersion/MinorVersion/PatchVersion`
* `build/windows_installer.nsi: PRODUCT_VERSION`

## License

PairMesh is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
