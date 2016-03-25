FROM golang:1.6-onbuild

MAINTAINER Marcus Ilgner <mail@marcusilgner.com>

ENV KEEDROP_REDIS "keedrop-redis.ud1pih.0001.euc1.cache.amazonaws.com:6379"
ENV KEEDROP_PORT  ":80"
EXPOSE 80
