# This Dockerfile is explicitly intended for use in running tests & builds of force.

FROM golang:1.6.2

RUN apt-get install -y mercurial

RUN go get -u github.com/Masterminds/glide && go get -u github.com/mitchellh/gox

# Mount the source code of the project directly into the image's GOPATH.
VOLUME /go/src/github.com/joist-engineering/force

WORKDIR /go/src/github.com/joist-engineering/force
