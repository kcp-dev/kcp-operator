# kcp-operator Documentation

[![Go Report Card](https://goreportcard.com/badge/github.com/kcp-dev/kcp-operator)](https://goreportcard.com/report/github.com/kcp-dev/kcp-operator)
[![GitHub](https://img.shields.io/github/license/kcp-dev/kcp-operator)](https://github.com/kcp-dev/kcp-operator/blob/main/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kcp-dev/kcp-operator?sort=semver)](https://github.com/kcp-dev/kcp-operator/releases/latest)
<!--[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkcp-dev%2Fkcp-operator.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkcp-dev%2Fkcp-operator?ref=badge_shield)-->

!!! warning
    While kcp-operator is usable, the project is still in an early state. Please only use it if you know what you are doing. We recommend against using it in production setups right now.

kcp-operator is a Kubernetes operator to deploy and run [kcp](https://github.com/kcp-dev/kcp) instances on a Kubernetes cluster. kcp is a horizontally scalable control plane for Kubernetes-like APIs.

## Features

- Create and update core components of a kcp setup (root shard, additional shards, front proxy)
- Support for multi-shard deployments of kcp
- Generate and refresh kubeconfigs for accessing kcp instances or specific shards

## Support Matrix

The table below marks known support of a kcp version in kcp-operator versions.

| kcp    | `main`             |
| ------ | ------------------ |
| `main` | :warning:          |
| 0.27.x | :white_check_mark: |

<small>While we try to support kcp's `main` branch, this support is best effort and should not be used for deploying actual kcp instances.</small>

## Contributing

We ❤️ our contributors! If you're interested in helping us out, please head over to our [Contributing](./contributing/index.md)
guide.

## Getting in touch

There are several ways to communicate with us:

- The [`#kcp-dev` channel](https://app.slack.com/client/T09NY5SBT/C021U8WSAFK) in the [Kubernetes Slack workspace](https://slack.k8s.io).
- Our mailing lists:
    - [kcp-dev](https://groups.google.com/g/kcp-dev) for development discussions.
    - [kcp-users](https://groups.google.com/g/kcp-users) for discussions among users and potential users.
- By joining the kcp-dev mailing list, you should receive an invite to our bi-weekly community meetings.
- See recordings of past community meetings on [YouTube](https://www.youtube.com/channel/UCfP_yS5uYix0ppSbm2ltS5Q).
- The next community meeting dates are available via our [CNCF community group](https://community.cncf.io/kcp/).
- Check the [community meeting notes document](https://docs.google.com/document/d/1PrEhbmq1WfxFv1fTikDBZzXEIJkUWVHdqDFxaY1Ply4) for future and past meeting agendas.
- Browse the [shared Google Drive](https://drive.google.com/drive/folders/1FN7AZ_Q1CQor6eK0gpuKwdGFNwYI517M?usp=sharing) to share design docs, notes, etc.
    - Members of the kcp-dev mailing list can view this drive.
