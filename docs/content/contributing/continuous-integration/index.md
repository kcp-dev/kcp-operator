# Continuous Integration (CI)

kcp-operator uses a combination of [GitHub Actions](https://help.github.com/en/actions/automating-your-workflow-with-github-actions) and
and [prow](https://github.com/kubernetes/test-infra/tree/master/prow) to automate the build process.

Here are the most important links:

- [.github/workflows/](https://github.com/kcp-dev/kcp-operator/blob/main/.github/workflows/) defines the Github Actions based jobs.
- [kcp-dev/kcp/.prow.yaml](https://github.com/kcp-dev/kcp-operator/blob/main/.prow.yaml) defines the prow based jobs.

## Running E2E tests locally

In order to run the E2E tests locally, you will need to setup cert-manager with the sample clusterissuer:

```sh
helm upgrade --install --namespace cert-manager --create-namespace --version v1.16.2 --set crds.enabled=true cert-manager jetstack/cert-manager
kubectl apply -n cert-manager --filename hack/ci/testdata/clusterissuer.yaml
```
