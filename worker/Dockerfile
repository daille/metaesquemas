FROM node:22-alpine
ENV NODE_ENV=production DATA_DIR=/data
WORKDIR /app
# La librería del catálogo es una dependencia local (file:), debe existir antes de npm ci
COPY package.json package-lock.json ./
COPY packages ./packages
RUN npm ci --omit=dev && npm cache clean --force
COPY src ./src
# Directorios de datos con el mismo propietario (uid 1000) que usa la API, para compartir los volúmenes
RUN mkdir -p /data/store /data/inbox /data/jobs /data/work && chown -R 1000:1000 /data /app
USER 1000:1000
CMD ["node", "src/main.js"]
