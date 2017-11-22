FROM alpine:3.6

RUN apk add -U bash ca-certificates curl py-pip
RUN pip install docker-compose

# Add S6-overlay to use S6 process manager
# https://github.com/just-containers/s6-overlay/#the-docker-way
ARG S6_VERSION=v1.21.0.0
ENV S6_BEHAVIOUR_IF_STAGE2_FAILS=2
RUN curl -sSL https://github.com/just-containers/s6-overlay/releases/download/${S6_VERSION}/s6-overlay-amd64.tar.gz | tar zxf -
COPY /rootfs /

ADD bin/version /bin/version
ADD webroot /var/www
ADD docker-compose.yml /docker-compose.yml
ADD config.json /root/.docker/config.json

EXPOSE 80
ENTRYPOINT ["/init"]
