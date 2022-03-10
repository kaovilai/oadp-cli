# CLI Suite for using [OpenShift API for Data Protection](redht.com/oadp)

Use by running 
```sh
docker run -it ghcr.io/kaovilai/oadp-cli:latest bash
```

Extends from https://github.com/kaovilai/openshift-cli by adding support for following commands:
- git
- velero
  - via alias to access velero binary on the velero server om the cluster
