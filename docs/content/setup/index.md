# Setup

## Requirements

- [cert-manager](https://cert-manager.io/) (see [Installing with Helm](https://cert-manager.io/docs/installation/helm/))

## Helm Chart

A Helm chart for kcp-operator is maintained in [kcp-dev/helm-charts](https://github.com/kcp-dev/helm-charts/tree/main/charts/kcp-operator). To install it, first add the Helm repository:

```sh
helm repo add kcp https://kcp-dev.github.io/helm-charts
```

And then install the chart:

```sh
helm install --create-namespace --namespace kcp-operator kcp-operator kcp/kcp-operator
```

For full configuration options, check out the Chart [values](https://github.com/kcp-dev/helm-charts/blob/main/charts/kcp-operator/values.yaml).

## Further Reading

{% include "partials/section-overview.html" %}
