# This Dockerfile builds all the dependencies needed by the monorepo, and should
# be used to build any of the follow-on services
FROM node:20.8.1-bullseye-slim as base
RUN apt-get update && apt-get install -y ca-certificates make
RUN npm install pnpm --global

COPY oplabs.crt /usr/local/share/ca-certificates/oplabs.crt
RUN chmod 644 /usr/local/share/ca-certificates/oplabs.crt \
  && update-ca-certificates

WORKDIR /opt/optimism/chain-mon

COPY ./ ./

RUN make build

ENV NODE_EXTRA_CA_CERTS=/usr/local/share/ca-certificates/oplabs.crt

ENTRYPOINT ["pnpm", "run"]