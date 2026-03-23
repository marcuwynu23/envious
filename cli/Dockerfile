FROM golang:1.21-alpine AS build
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/envious .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /out/envious /usr/local/bin/envious
ENTRYPOINT ["envious"]
CMD ["--help"]

