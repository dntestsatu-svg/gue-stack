FROM node:22-alpine AS builder

WORKDIR /app

ARG VITE_SITE_URL
ARG VITE_API_BASE_URL
ARG VITE_GTM_ID=
ARG VITE_CSRF_HEADER_NAME=X-CSRF-Token
ARG VITE_CSRF_COOKIE_NAME=csrf_token

ENV VITE_SITE_URL=$VITE_SITE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_GTM_ID=$VITE_GTM_ID
ENV VITE_CSRF_HEADER_NAME=$VITE_CSRF_HEADER_NAME
ENV VITE_CSRF_COOKIE_NAME=$VITE_CSRF_COOKIE_NAME

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
