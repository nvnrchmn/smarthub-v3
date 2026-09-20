# Smarthub V3 Backend — Dockerfile (Multi-stage)
#
# Build: docker build -t smarthub-api .
# Run local:  docker run -p 8082:8082 --env-file .env smarthub-api
# Prod compose:  gunakan docker-compose.prod.yml
#
# Stage 1: Builder (full deps, devDependencies)
FROM oven/bun:latest AS builder
WORKDIR /app
COPY package.json bun.lockb* ./
RUN bun install  # tanpa frozen-lockfile, belum ada lockfile
COPY . .
RUN bun build src/index.ts --outdir=dist --format=esm --target=bun

# Stage 2: Runtime (hanya production deps)
FROM oven/bun:latest AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY package.json bun.lockb* ./
RUN bun install --production  # tanpa frozen-lockfile, belum ada lockfile

# Salin hasil build + file penting
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/src/db/migrations ./src/db/migrations

# Port: 8082 (smarthub-api)
EXPOSE 8082
USER bun
CMD ["bun", "run", "dist/index.js"]