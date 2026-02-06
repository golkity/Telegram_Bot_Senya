FROM ubuntu:latest
LABEL authors="finnik"

ENTRYPOINT ["top", "-b"]