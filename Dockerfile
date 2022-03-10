FROM ghcr.io/kaovilai/openshift-cli:v4.9.9

RUN apk add --update --no-cache git bash
RUN touch ~/.bashrc
RUN echo $'alias velero=\'oc -n openshift-adp exec deployment/velero -c velero -it -- ./velero\'' >> ~/.bashrc
RUN echo $'cd /root' >> ~/.bashrc