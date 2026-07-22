# GitLab setup

This repository is configured for GitLab CI/CD, GitLab security scans, container registry builds, and Kubernetes deployment through the GitLab Agent.

## Enabled in the repository

- Go test and vet with PostgreSQL service
- Docker image build and push to GitLab Container Registry
- SAST
- Secret Detection
- Dependency Scanning
- Container Scanning
- Acceptance test
- k6 load test with `p95 < 200ms` and `max < 200ms`
- Manual Kubernetes deployment job

## Required GitLab variables

Set these in `Settings > CI/CD > Variables`:

```text
DATABASE_URL
```

`DATABASE_URL` is used by the manual Kubernetes deploy job to create the `warehouse-routing` Kubernetes secret.

The runner that executes Docker jobs must support Docker-in-Docker with privileged mode enabled. The pipeline uses:

```text
DOCKER_HOST=tcp://docker:2375
DOCKER_TLS_CERTDIR=
COMPOSE_PROJECT_NAME=warehouse-routing
```

GitLab provides these registry variables automatically:

```text
CI_REGISTRY
CI_REGISTRY_USER
CI_REGISTRY_PASSWORD
CI_REGISTRY_IMAGE
```

## Kubernetes Agent

The repo-side agent config is:

```text
.gitlab/agents/warehouse-routing/config.yaml
```

Register an agent named:

```text
warehouse-routing
```

Then install that agent in the target Kubernetes cluster using the command GitLab shows in:

```text
Operate > Kubernetes clusters > Connect a cluster
```

The deploy job uses this context:

```text
rqzbeh/warehouse-routing:warehouse-routing
```

The deploy job is manual on `main`. It applies:

```text
k8s/deployment.yaml
k8s/service.yaml
k8s/hpa.yaml
```

It also replaces the local image name with the GitLab Container Registry image built for the current commit.

The deploy job creates these Kubernetes secrets:

```text
warehouse-routing
gitlab-registry
```

`warehouse-routing` stores the application `DATABASE_URL`. `gitlab-registry` lets the cluster pull the private GitLab Container Registry image.
