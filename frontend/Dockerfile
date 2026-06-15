ARG NODE_IMAGE=node:22-alpine3.23
ARG NGINX_IMAGE=nginxinc/nginx-unprivileged:1.29.4-alpine

FROM ${NODE_IMAGE} AS builder

WORKDIR /app

ARG VUE_APP_API_URL=http://localhost:8081
ENV VUE_APP_API_URL=${VUE_APP_API_URL}
ENV NODE_OPTIONS=--openssl-legacy-provider

COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .

RUN npm run build

FROM ${NGINX_IMAGE}

USER root

RUN apk upgrade --no-cache

COPY --from=builder --chown=101:101 /app/dist /usr/share/nginx/html
COPY --chown=101:101 nginx.conf /etc/nginx/conf.d/default.conf

USER 101:101

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["nginx", "-g", "daemon off;"]
