FROM alpine:latest AS GOLANG
RUN apk add git go
ADD . /src
WORKDIR /src
ARG USER
ARG TOKEN
RUN go env -w GOPRIVATE=github.com/deliveryblueprints/*
RUN git config --global url."https://${USER}:${TOKEN}@github.com".insteadOf  "https://github.com"
RUN go get -d
RUN go build -o /tmp/plugin-jenkins-master
RUN ls -lrt /tmp

FROM alpine:latest
WORKDIR /app/
COPY --from=GOLANG /tmp/plugin-jenkins-master /app/plugin-jenkins-master
ENTRYPOINT ["/app/plugin-jenkins-master"]