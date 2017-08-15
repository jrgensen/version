FROM alpine:3.6

RUN apk add -U ca-certificates

ADD bin/version /bin/version
ADD webroot /var/www

EXPOSE 80
CMD /bin/version -port=80 -hosts=$VERSION_HOSTS
