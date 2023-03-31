# CLI Suite for using [OpenShift API for Data Protection](https://github.com/openshift/oadp-operator)

Use by running 
```sh
docker run -it ghcr.io/kaovilai/oadp-cli:latest bash
```

Extends from https://catalog.redhat.com/software/containers/openshift4/ose-cli by adding support for following commands:
- git
- velero
  - via alias to access velero binary on the velero server on the cluster
