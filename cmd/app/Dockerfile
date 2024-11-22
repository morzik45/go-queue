FROM golang:1.22-alpine as builder-go
LABEL stage=builder
WORKDIR /usr/src/app
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
 apk add --no-cache upx ca-certificates tzdata
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o server ./cmd/app &&\
 upx --best server -o _upx_server && \
 mv -f _upx_server server

FROM scratch as runner
COPY --from=builder-go /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=builder-go /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder-go /usr/src/app/server /opt/app/
CMD ["/opt/app/server"]
