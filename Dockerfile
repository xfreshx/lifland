FROM golang:1.12

LABEL maintainer="Oleksii Lykhosherstov <alexey.likhosherstov@dataart.com>"

WORKDIR $GOPATH/src/github.com/xfreshx/lifland

COPY . .

RUN go get -d -v ./...

RUN go install -v ./...

EXPOSE 8080

CMD ["lifland"]