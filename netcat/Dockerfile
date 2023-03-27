# https://www.tecmint.com/linux-distributions-for-containers/
FROM alpine:latest

RUN apk add --update --no-cache netcat-openbsd
COPY ./netcatTest.sh /
RUN chmod +x ./netcatTest.sh
CMD ["/bin/sh", "./netcatTest.sh"]
