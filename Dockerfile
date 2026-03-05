FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY tsconfig.json ./
COPY src/server ./src/server

RUN npx tsc --project tsconfig.json

# --- Production image ---
FROM node:22-alpine

RUN apk add --no-cache dumb-init

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY --from=builder /app/dist/server ./dist/server

ENV NODE_ENV=production
ENV IMITSU_PORT=3100
ENV IMITSU_DB_PATH=/data/imitsu.db

RUN mkdir -p /data && chown node:node /data

EXPOSE 3100

VOLUME /data

USER node

ENTRYPOINT ["dumb-init", "--"]
CMD ["node", "dist/server/index.js"]
