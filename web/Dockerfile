# Use an official Node.js runtime as the base image
FROM node:18-alpine AS build

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
FROM nginx:alpine

# Copy the built React app from the previous stage
COPY --from=build /app/dist /usr/share/nginx/html

# Expose port 80 (nginx default listen port)
EXPOSE 80

# Start Nginx server
CMD ["nginx", "-g", "daemon off;"]
