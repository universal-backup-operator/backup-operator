Getting started
===

## Installation

First, you have to install an operator into your cluster.

!!! info "Dependency"

    Backup operator depends on [Cert Manager](https://cert-manager.io). Please, follow the [official documentation](https://cert-manager.io/docs/installation/) to install it first.

=== "Flux"
 
    ```yaml title="HelmRepository"
    apiVersion: source.toolkit.fluxcd.io/v1
    kind: HelmRepository
    metadata:
      name: backup-operator
    spec:
      interval: 1h0m0s
      provider: generic
      url: https://helm-charts.backup-operator.io/
    ```

    ```yaml title="HelmRelease"
    apiVersion: helm.toolkit.fluxcd.io/v2
    kind: HelmRelease
    metadata:
      name: backup-operator
    spec:
      releaseName: backup-operator
      interval: 1h0m0s
      timeout: 5m
      # Optional
      # dependsOn:
      # - name: cert-manager
      chart:
        spec:
          chart: backup-operator
          # Optional
          # version: "<2.0.0"
          sourceRef:
            kind: HelmRepository
            name: backup-operator
      install:
        crds: CreateReplace
      upgrade:
        crds: CreateReplace
      values:
        nameOverride: backup-operator
        fullnameOverride: backup-operator
        replicaCount: 2
        resources:
          limits:
            memory: 150Mi
          requests:
            cpu: 20m
            memory: 150Mi
        pdb:
          enabled: true
        metrics:
          enabled: false
          serviceMonitor:
            create: false
    ```

=== "Helm"

    ```shell title="Add Helm repository"
    helm repo add backup-operator https://helm-charts.backup-operator.io/
    ```
    
    ```shell title="Install Helm chart"
    helm install backup-operator backup-operator/backup-operator
    ```

    Also we have ArtifactHub [page](https://artifacthub.io/packages/helm/backup-operator/backup-operator?modal=install).
