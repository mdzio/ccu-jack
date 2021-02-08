FROM golang:latest AS builder
#FROM golang:1.15-alpine AS builder
ADD . /src
RUN cd /src/build && go build -o ccu-jack .

FROM scratch as run
#WORKDIR /app
COPY --from=builder /src/build/ccu-jack /
ENTRYPOINT ./ccu-jack
# CMD ["./ccu-jack"]