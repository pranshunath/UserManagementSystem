# =============================================================
# Stage 1: Build production React + TypeScript + Vite assets
# =============================================================
FROM node:22-alpine AS builder

WORKDIR /app

# Cache package dependencies
COPY package*.json ./
RUN npm ci

# Copy full frontend source code
COPY . .

# Environment variables for production build
# Relative paths route seamlessly through the Nginx reverse proxy
ENV VITE_API_URL=/api/v1
ENV VITE_HEALTH_URL=/health

# Run TypeScript check and production bundle compilation
RUN npm run build

# =============================================================
# Stage 2: Serve compiled assets using high-performance Nginx
# =============================================================
FROM nginx:alpine

# Remove default static files
RUN rm -rf /usr/share/nginx/html/*

# Copy compiled static assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy customized Nginx reverse proxy configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# Healthcheck validating Nginx is responding on port 80
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
