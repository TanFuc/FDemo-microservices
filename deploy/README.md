# NexusCommerce Enterprise Infrastructure & Deployment Guide

Welcome to the centralized platform engineering and deployment documentation for **NexusCommerce**. All infrastructure-as-code (IaC), container orchestration, continuous integration/continuous delivery (CI/CD), Kubernetes manifests, and observability assets are consolidated within this directory.

---

## Architecture Overview

```
                                      +---------------------------------------------+
                                      |            GitHub Actions CI/CD             |
                                      |  (Matrix Test, Vet, Docker Build, Linting)  |
                                      +----------------------+----------------------+
                                                             |
                                           GitOps Automated Sync (ArgoCD)
                                                             |
                                                             v
+-------------------------------------------------------------------------------------------------------------------+
| Kubernetes Cluster (Production / Staging)                                                                         |
|                                                                                                                   |
|   Ingress Controller (TLS Termination, Nginx / Traefik)                                                           |
|          |                                                                                                        |
|          v                                                                                                        |
|   [API Gateway] (Rate Limiting, Reverse Proxy, Route Authentication)                                              |
|          |                                                                                                        |
|   +------+-------------------+-------------------+-------------------+--------------------+                       |
|   |                          |                   |                   |                    |                       |
|   v                          v                   v                   v                    v                       |
| [Auth Service]       [Order Service]     [Inventory Svc]     [Payment Svc]        [Catalog Svc] ...               |
|   |                          |                   |                   |                    |                       |
|   +-------------+------------+---------+---------+---------+---------+----------+---------+                       |
|                 |                      |                   |                    |                                 |
|                 v                      v                   v                    v                                 |
|          Event Bus (NATS)       PostgreSQL Pools      Redis Clusters       ClickHouse OLAP                        |
|                                                                                                                   |
|   Observability Sidecars & DaemonSets:                                                                            |
|   - Prometheus (Metrics scraping :9090)                                                                           |
|   - Grafana Tempo (OTLP Traces :4317/:4318)                                                                       |
|   - Grafana Loki (Log shipping :3100)                                                                             |
+-------------------------------------------------------------------------------------------------------------------+
```

---

## Directory Structure

```
deploy/
├── README.md                           # This platform engineering manual
├── compose/                            # Docker Compose environments
│   ├── .env.example                    # Production & local environment variable blueprint
│   ├── docker-compose.databases.yml    # Data plane: 6x Postgres, 6x MongoDB, Redis, ClickHouse, MinIO, NATS
│   ├── docker-compose.prod.yml         # Full application plane: 15 microservices + API Gateway
│   └── docker-compose.observability.yml# Observability plane: Prometheus, Grafana, Tempo, Loki
├── helm/                               # Cloud-Native Kubernetes Helm Charts
│   └── nexus-service/                  # Parameterized microservice chart
│       ├── Chart.yaml                  # Chart metadata & API version
│       ├── values.yaml                 # Base configuration
│       ├── values-staging.yaml         # Staging overrides (single replica, debug logs)
│       ├── values-prod.yaml            # Production overrides (HPA, PDB, strict limits)
│       └── templates/                  # Deployment, Service, Ingress, HPA, PDB, ConfigMap
├── argocd/                             # GitOps Declarative Deployment Manifests
│   ├── application-root.yaml           # App-of-apps root orchestrator
│   ├── app-staging.yaml                # Staging environment ArgoCD application
│   └── app-prod.yaml                   # Production environment ArgoCD application
├── ci/                                 # CI/CD Workflow Blueprints
│   └── github-actions/
│       ├── ci.yml                      # Parallel testing, vet, and config validation
│       └── docker-build.yml            # Multi-architecture container build & push pipeline
└── observability/                      # Telemetry & Monitoring Configurations
    ├── prometheus/
    │   └── prometheus.yml              # Prometheus scrape jobs & targets
    └── tempo/
        └── tempo.yaml                  # Distributed tracing collector configuration
```

---

## 1. Local Development (Docker Compose)

The local stack is separated into modular layers so developers can start only what they need:

### Layer A: Data Plane & Message Brokers
Launches PostgreSQL (with 6 databases initialized), MongoDB (with 6 databases), Redis, ClickHouse, MinIO, Elasticsearch, Kibana, NATS JetStream, and RabbitMQ:

```bash
# Using the root Makefile
make up-db

# Or using docker compose directly
docker compose -f deploy/compose/docker-compose.databases.yml up -d
```

### Layer B: Observability Plane
Launches Prometheus, Grafana, Tempo, and Loki:

```bash
# Using Makefile
make up-obs

# Access Dashboards:
# - Prometheus: http://localhost:9090
# - Grafana:    http://localhost:3001 (Credentials: admin / nexus_grafana_admin)
# - Tempo:      http://localhost:3200
```

### Layer C: Full Application Simulation
Runs all 15 compiled microservices connected via the internal bridge network:

```bash
# Copy and configure environment variables
cp deploy/compose/.env.example deploy/compose/.env

# Launch full microservices stack
make up-prod
```

---

## 2. Kubernetes Deployment (Helm Charts)

The unified `nexus-service` Helm chart is designed with high-availability enterprise patterns:
- **Horizontal Pod Autoscaling (HPA)** based on target CPU (65%) and Memory (75%) utilization.
- **Pod Disruption Budgets (PDB)** ensuring minimum service availability during node upgrades or evictions.
- **Non-root Security Contexts** (`runAsNonRoot: true`, `readOnlyRootFilesystem: true`).
- **Prometheus Metric Annotations** for automatic metrics scraping.

### Helm Commands
```bash
# Validate chart syntax
helm lint deploy/helm/nexus-service

# Dry-run render templates for staging
helm template nexus-order deploy/helm/nexus-service -f deploy/helm/nexus-service/values-staging.yaml

# Dry-run render templates for production
helm template nexus-order deploy/helm/nexus-service -f deploy/helm/nexus-service/values-prod.yaml

# Deploy to cluster
helm upgrade --install nexus-order deploy/helm/nexus-service \
  --namespace nexus-prod \
  --create-namespace \
  -f deploy/helm/nexus-service/values-prod.yaml
```

---

## 3. GitOps Continuous Delivery (ArgoCD)

NexusCommerce uses the **App-of-Apps** pattern with ArgoCD:
1. `deploy/argocd/application-root.yaml`: Tracks the repository and synchronizes application definitions.
2. `deploy/argocd/app-staging.yaml`: Deploys services to the `nexus-staging` namespace tracked against the `develop` branch.
3. `deploy/argocd/app-prod.yaml`: Deploys high-availability workloads to `nexus-prod` tracked against the `main` branch.

To bootstrap ArgoCD:
```bash
kubectl apply -f deploy/argocd/application-root.yaml
```

---

## 4. Continuous Integration (GitHub Actions)

Located at `.github/workflows/ci.yml` (blueprint at `deploy/ci/github-actions/ci.yml`), our CI pipeline executes:
1. **Parallel Go Verification Matrix**: Downloads dependencies, vets code, and runs tests with race detector across all Go modules.
2. **Infrastructure Validation**: Runs `docker compose config` against all compose files to prevent schema regressions.
3. **Helm Linting**: Ensures all Kubernetes templates and values files conform to standards.

---

## 5. Observability & SRE Metrics

NexusCommerce implements the **RED Method** (Rate, Errors, Duration):
- **Prometheus** scrapes `/metrics` endpoints from all Go microservices.
- **Tempo** aggregates distributed traces through OpenTelemetry (OTLP gRPC `:4317` and HTTP `:4318`).
- **Loki** ingests container logs.
