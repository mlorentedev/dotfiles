---
generated: true
generated_from: 00_meta/skills/docker/SKILL.md
generated_sha: 2a019c5021743efd
id: docker-skill
type: skill
status: active
created: "2026-05-30"
owner: manu
name: docker
description: Use when containerizing applications, setting up local development environments, or creating multi-service Docker deployments.
---

# Docker Configuration

Generate `Dockerfile` and `docker-compose.yml`.

## Requirements

| Aspect | Rule |
|--------|------|
| Base image | Specific version, never `latest`. Prefer slim/alpine/distroless |
| User | Non-root (`USER appuser`) |
| Build | Multi-stage to minimize final image |
| Layers | Dependencies before code for caching |
| Secrets | Via environment variables, never in image |
| Health | `HEALTHCHECK` instruction required |

## Dockerfile Template

```dockerfile
# Build
FROM python:3.12-slim-bookworm AS builder
WORKDIR /build
COPY requirements.txt .
RUN pip wheel --no-cache-dir -r requirements.txt -w /wheels

# Production
FROM python:3.12-slim-bookworm
WORKDIR /app
RUN useradd -r -s /bin/false appuser
COPY --from=builder /wheels /wheels
RUN pip install --no-cache-dir /wheels/* && rm -rf /wheels
COPY --chown=appuser:appuser . .
USER appuser
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:8000/health || exit 1
CMD ["python", "-m", "app"]
```

## docker-compose.yml Template

```yaml
services:
  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      DATABASE_URL: postgresql://user:pass@db:5432/app
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: app
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d app"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

## Output

Both files with comments explaining non-obvious choices.
