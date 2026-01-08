# ==============================================================================
# PLM Frontend Dockerfile (Next.js)
# ==============================================================================
# Production-ready Next.js with standalone output
# ==============================================================================

FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build with standalone output
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ==============================================================================
# Production Runtime
# ==============================================================================
FROM node:20-alpine

WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S plm_group && \
    adduser -u 1001 -S plm_user -G plm_group

# Copy standalone build
COPY --from=builder --chown=plm_user:plm_group /app/.next/standalone ./
COPY --from=builder --chown=plm_user:plm_group /app/.next/static ./.next/static
COPY --from=builder --chown=plm_user:plm_group /app/public ./public

# Switch to non-root user
USER plm_user

# Environment
ENV NODE_ENV=production
ENV PORT=3000

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:3000 || exit 1

# Start server
CMD ["node", "server.js"]
