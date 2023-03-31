FROM registry.redhat.io/openshift4/ose-cli

RUN dnf install -y git bash
RUN touch ~/.bashrc
RUN echo $'alias velero=\'oc -n openshift-adp exec deployment/velero -c velero -it -- ./velero\'' >> ~/.bashrc
RUN echo $'cd /root' >> ~/.bashrc
CMD [ "bash" ]