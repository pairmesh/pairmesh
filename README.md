![](docs/images/logo.svg)
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

- How to build PairMesh
  ```shell
  # Linux/macOS Terminal
  make pairmesh
  
  # Windows PowerShell
  go build -o bin/PairMesh.exe  -ldflags "-s -w -H=windowsgui -X github.com/pairmesh/pairmesh/version.GitHash=$(git describe --no-match --always --dirty) -X github.com/pairmesh/pairmesh/version.GitBranch=$(git rev-parse --abbrev-ref HEAD)" ./cmd/pairmesh
  ```
- **Use [PairMesh](https://www.pairmesh.com) managed control plane service.**
- Self-hosted PairMesh control plane.

## Community

You can join these discussion forum and chats to discuss and ask PairMesh related questions:

- [PairMesh Discussion](https://github.com/pairmesh/pairmesh/discussions)

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

- [How to setup development environment](docs/guide/dev-guide.md)

Any other question? Reach out to the [PairMesh Discussion](https://github.com/pairmesh/pairmesh/discussions) forum to get help!

## Architecture

The following diagram shows the overall architecture of PairMesh, where the PairMesh node is an application installed on the end device, responsible for managing the local virtual NIC device and discovering information about other nodes from the Control Plane service, as well as establishing P2P communication connections with other nodes, encrypting and decrypting data and processing traffic data in the network.

![Architecture](./docs/images/architecture.svg)
## License

PairMesh is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
