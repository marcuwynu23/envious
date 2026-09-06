FROM golang:1.23-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-X envious-web/internal/version.Version=${VERSION} -X envious-web/internal/version.Commit=${COMMIT} -X envious-web/internal/version.BuildDate=${BUILD_DATE}" -o envious-server ./cmd/server

FROM alpine:3.19
WORKDIR /srv
COPY --from=build /app/envious-server /usr/local/bin/envious-server
EXPOSE 8080
ENV PORT=8080
CMD ["envious-server"]

