# Setup

## Requirements

- [cert-manager](https://cert-manager.io/)

## Helm Chart

A Helm chart for kcp-operator is maintained in [kcp-dev/helm-charts](https://github.com/kcp-dev/helm-charts/tree/main/charts/kcp-operator). To install it, first add the Helm repository:

```sh
helm repo add kcp https://kcp-dev.github.io/helm-charts
```

And then install the chart:

```sh
helm upgrade --install --create-namespace --namespace kcp-operator kcp-operator kcp/kcp-operator
```

## Further Reading

{% include "partials/section-overview.html" %}
