# Stage 1: Build Angular app
FROM node:22-alpine AS builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build -- --configuration production

# Stage 2: Serve with nginx
FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY --from=builder /app/ui/dist/ui/browser /usr/share/nginx/html
COPY docker/ui/nginx.conf.template /etc/nginx/nginx.conf.template
COPY --chmod=0755 docker/ui/40-init-nginx-config.sh /docker-entrypoint.d/40-init-nginx-config.sh
EXPOSE 8080
