# kcp-dev/kcp-operator

[![Go Report Card](https://goreportcard.com/badge/github.com/kcp-dev/kcp-operator)](https://goreportcard.com/report/github.com/kcp-dev/kcp-operator)
[![GitHub](https://img.shields.io/github/license/kcp-dev/kcp-operator)](https://github.com/kcp-dev/kcp-operator/blob/main/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kcp-dev/kcp-operator?sort=semver)](https://github.com/kcp-dev/kcp-operator/releases/latest)
<!--[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkcp-dev%2Fkcp-operator.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkcp-dev%2Fkcp-operator?ref=badge_shield)-->

> [!WARNING]
> While kcp-operator is usable, the project is still in an early state. Please only use it if you know what you are doing. We recommend against using it in production setups right now.

kcp-operator is a Kubernetes operator to deploy and run [kcp](https://github.com/kcp-dev/kcp) instances on a Kubernetes cluster. kcp is a horizontally scalable control plane for Kubernetes-like APIs.

## Features

- [x] Create and update core components of a kcp setup (root shard, additional shards, front proxy)
- [x] Support for multi-shard deployments of kcp
- [ ] Support for a dedicated cache-server deployment not embedded in the root shard
- [x] Generate and refresh kubeconfigs for accessing kcp instances or specific shards
- [ ] Cross-namespace/-cluster setups of a multi-shard kcp deployment

## Support Matrix

The table below marks known support of a kcp version in kcp-operator versions.

| kcp    | `main`             |
| ------ | ------------------ |
| `main` | :warning: [^1]     |
| 0.27.x | :white_check_mark: |

[^1]: While we try to support kcp's `main` branch, this support is best effort and should not be used for deploying actual kcp instances.

## Contributing

Thanks for taking the time to start contributing! Please check out our [contributor documentation](https://docs.kcp.io/kcp-operator/main/contributing).

### Before You Start

* Please familiarize yourself with the [Code of Conduct][4] before contributing.
* See [our contributor documentation][2] for instructions on the developer certificate of origin that we require.

### Pull Requests

* We welcome pull requests. Feel free to dig through the [issues][1] and jump in.

## Changelog

See [the list of releases][3] to find out about feature changes.

## License

This project is licensed under [Apache-2.0](./LICENSE).

[1]: https://github.com/kcp-dev/kcp-operator/issues
[2]: https://docs.kcp.io/kcp/main/contributing/getting-started/#developer-certificate-of-origin-dco
[3]: https://github.com/kcp-dev/kcp-operator/releases
[4]: https://github.com/kcp-dev/kcp/blob/main/code-of-conduct.md
