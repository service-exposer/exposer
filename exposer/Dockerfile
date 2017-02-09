FROM golang:1.7-onbuild

ENV FORWARD_KEY 123456

CMD go-wrapper run forward-server -k ${FORWARD_KEY} -a :9000 --protocal ws
EXPOSE 9000
