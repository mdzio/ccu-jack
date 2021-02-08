# V0.1
# first working Dockerfile
# build the image with
#   docker build -t ccu-jack:latest .
# Workaround: put your config in directory "wd" to mount it into container 
# run it with 
#   docker run --rm  -v "$PWD"/wd/ccu-jack.cfg:/go/src/app/ccu-jack.cfg:ro ccu-jack:latest
#FROM golang:latest # alpine is much smaller but might have problems
FROM golang:1.15-alpine
WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go build -o ccu-jack .

CMD ["./ccu-jack"]