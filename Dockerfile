FROM busybox
MAINTAINER Ian Quick <ian.quick@gmail.com>
ADD es-k8s /
ENTRYPOINT ["/es-k8s"]
