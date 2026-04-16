# Skill: Deployment

How to safely deploy changes to production.

## Pre-deploy checklist

Before any deployment, verify every item:

### 1. Build verification

```
# Clean build from scratch
docker build --no-cache -t app:candidate .

# Run the test suite inside the container
docker run --rm app:candidate npm test

# Check image size (keep it under 500 MB)
docker images app:candidate --format "{{.Size}}"
```

### 2. Staging validation

```
# Deploy to staging first
docker run -d --name staging -p 3001:3000 app:candidate

# Smoke test
curl -f http://localhost:3001/health || echo "HEALTH CHECK FAILED"

# Run integration tests against staging
npm run test:integration -- --target=http://localhost:3001

# Tear down staging
docker stop staging && docker rm staging
```

### 3. Deploy

```
# Tag the release
git tag -a v$(date +%Y%m%d-%H%M) -m "Release $(date +%Y%m%d)"

# Deploy (replace with your actual deploy command)
docker tag app:candidate app:latest
```

### 4. Post-deploy verification

```
# Verify the deploy
curl -f http://localhost:3000/health

# Watch logs for errors (first 60 seconds)
docker logs -f --since 1m app
```

### 5. Rollback plan

If anything goes wrong:

```
# Revert to previous image
docker stop app && docker rm app
docker run -d --name app -p 3000:3000 app:previous
```

## Deployment principles

- **Never deploy on Friday.** Seriously.
- **Staging first, always.** No exceptions, no "it's a small change."
- **One deploy at a time.** Wait for the current deploy to stabilize before starting another.
- **Log the result.** Every deploy goes into the journal, pass or fail.
