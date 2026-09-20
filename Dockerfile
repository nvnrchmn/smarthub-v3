# Dockerfile — multi-stage (backend Hono)
FROM oven/bun:latest AS base
WORKDIR /app
RUN bun prisma generate

# Install dependencies
FROM base AS install
COPY backend/package.json backend/bun.lockb ./
RUN bun install --frozen-lockfile --prod

# Build
FROM base AS build
COPY backend/ ./
RUN bun run build

# Run
FROM base AS final
COPY --from=install /app/node_modules /app/node_modules
COPY --from=build /app/dist /app/dist
USER bun
EXPOSE 4000
CMD ["bun", "run", "start"]