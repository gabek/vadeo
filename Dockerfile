FROM golang:1.17-alpine as builder
RUN mkdir /build 
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o vadeo .

FROM jrottenberg/ffmpeg:4.1-alpine
COPY --from=builder /build/vadeo /app/
COPY --from=builder /build/logo.png /app/
COPY --from=builder /build/polentical-neon.ttf /app/
COPY --from=builder /build/background.mp4 /app/
WORKDIR /app
ENTRYPOINT ["./vadeo"]