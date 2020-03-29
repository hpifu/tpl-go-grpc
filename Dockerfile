FROM centos:centos7

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" >> /etc/timezone

COPY docker/tpl-go-grpc /var/docker/tpl-go-grpc
RUN mkdir -p /var/docker/tpl-go-grpc/log

EXPOSE 7060

WORKDIR /var/docker/tpl-go-grpc
CMD [ "bin/echo", "-c", "configs/echo.json" ]
