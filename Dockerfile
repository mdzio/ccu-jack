FROM golang:1.15-alpine AS builder
ADD . /src
RUN cd /src/build && go build -v -o ccu-jack .

FROM scratch as run
WORKDIR /app
COPY --from=builder /src/build/ccu-jack /app/
ENTRYPOINT ./ccu-jack
# CMD ["./ccu-jack"]