# Frontend Dockerfile
FROM node:18 as builder
WORKDIR /app

# Copy package.json and package-lock.json
COPY package.json package-lock.json ./

# Install dependencies
RUN npm install

# Copy the source code and public directory
COPY ./src ./src
COPY ./public ./public

# Build the React app
RUN npm run build

# Use nginx to serve the built app
FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
COPY src/chartpress.yaml /usr/share/nginx/html/

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
