FROM golang:1.9.1

RUN go get github.com/cespare/reflex
ADD . /go/src/github.com/Scalingo/sand
WORKDIR /go/src/github.com/Scalingo/sand
EXPOSE 9999
RUN go install
CMD /go/bin/sand
