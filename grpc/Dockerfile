FROM golang:1.17-alpine as build

WORKDIR /app
ENV GO111MODULE=on

RUN apk --no-cache add ca-certificates

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /bin/app cmd/grpc.go

FROM scratch
COPY --from=build /bin/app /
ENTRYPOINT ["/app"]
