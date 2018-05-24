FROM centos:7

MAINTAINER APPUiO Team

RUN rpm -ihv https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm && \
    yum -y install openssl jq && \
    yum clean all && \
    mkdir -p /srv/.well-known/acme-challenge /var/lib/letsencrypt && \
    chmod 775 /srv/.well-known/acme-challenge && \
    cd /usr/local/bin && \
    curl -O https://console.appuio.ch/console/extensions/clients/linux/oc && \
    chmod 755 /usr/local/bin/oc && \
    ln -s /go/src/github.com/appuio/letsencrypt /usr/local/letsencrypt

ADD . /go/src/github.com/appuio/letsencrypt/

RUN export GOPATH=/go && \
    yum -y install golang-bin && \
    cd /usr/local/letsencrypt && \
    go install github.com/appuio/letsencrypt && \
    yum -y history undo last && \
    yum clean all && \
    chmod g+w /usr/local/letsencrypt/bin/*

USER 1001

ENV HOME=/var/lib/letsencrypt

EXPOSE 8080

CMD ["/go/bin/letsencrypt"]
