FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /app/app .



FROM alpine:3.23

RUN apk --no-cache add ca-certificates

#===============================#
#        OCI Metadata           #
#===============================#

ARG IMAGE_TAG
LABEL org.opencontainers.image.version=$IMAGE_TAG

#===============================#
#        Build Metadata         #
#===============================#

ENV IMAGE_TAG=$IMAGE_TAG

#===============================#
#   Application Configuration   #
#===============================#

WORKDIR /app

COPY --from=builder /app/app .

CMD ["./app"]