# syntax=docker/dockerfile:1.7
# SPDX-License-Identifier: Apache-2.0
FROM node:20.18.3-bookworm-slim@sha256:ffc11dbf16dd0abcbb7b837410601b4d5592db2d03741e13a4a5336ab74d7ccb AS build

# Set the working directory in the container
WORKDIR /app

# Copy the manifest + lockfile first so the install layer caches on deps alone
COPY package.json package-lock.json ./

# Install exactly what the lockfile pins (reproducible; fails on drift)
RUN npm ci --no-audit --no-fund

# Copy the rest of the application code
COPY . .

# Build the React app
RUN npm run build

# Use a lightweight web server to serve the built app
FROM nginx:1.27.4-alpine@sha256:4ff102c5d78d254a6f0da062b3cf39eaf07f01eec0927fd21e219d0af8bc0591
ARG VCS_REF=unknown
ARG SOURCE_URL=https://github.com/kriipke/chartpress
LABEL org.opencontainers.image.source=$SOURCE_URL \
      org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.licenses=Apache-2.0

# Copy the built React app from the previous stage
COPY --from=build /app/dist /usr/share/nginx/html

# Expose port 80 (nginx default listen port)
EXPOSE 80

# Start Nginx server
CMD ["nginx", "-g", "daemon off;"]
