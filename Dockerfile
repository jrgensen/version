FROM busybox

ADD bin/version /bin/version

EXPOSE 80
CMD /bin/version -port 80
