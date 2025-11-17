# ArgoCD Applications

This directory contains ArgoCD Application manifests for GitOps deployment.

## Setup

### 1. Install ArgoCD on Production Cluster

```bash
# Create argocd namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available deployment/argocd-server -n argocd --timeout=300s

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d; echo

# Port forward to access UI
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Access at: https://localhost:8080
# Username: admin
# Password: (from above command)
```

### 2. Configure Repository Access

If your repository is private, add it to ArgoCD:

```bash
# Via UI:
# Settings → Repositories → Connect Repo → Via HTTPS or SSH

# Via CLI:
argocd repo add https://github.com/yourusername/welcomebot-template2 \
  --username YOUR_USERNAME \
  --password YOUR_GITHUB_TOKEN

# Or with SSH:
argocd repo add git@github.com:yourusername/welcomebot-template2.git \
  --ssh-private-key-path ~/.ssh/id_rsa
```

### 3. Deploy ArgoCD Applications

**Update Repository URL First:**
```bash
# Edit application-staging.yaml
vim deployments/argocd/application-staging.yaml
# Change: repoURL: https://github.com/yourusername/welcomebot-template2

# Edit application-production.yaml
vim deployments/argocd/application-production.yaml
# Change: repoURL: https://github.com/yourusername/welcomebot-template2
```

**Apply the applications:**
```bash
# Apply staging application
kubectl apply -f deployments/argocd/application-staging.yaml

# Apply production application
kubectl apply -f deployments/argocd/application-production.yaml

# Check status
kubectl get application -n argocd

# Watch sync status
argocd app get welcomebot-staging
argocd app get welcomebot-production
```

## Applications

### welcomebot-staging

- **Path:** `deployments/overlays/staging`
- **Namespace:** `welcomebot-staging`
- **Sync:** Automatic on Git changes
- **Self-Heal:** Enabled (auto-corrects manual changes)
- **Prune:** Enabled (deletes resources removed from Git)

### welcomebot-production

- **Path:** `deployments/overlays/production`
- **Namespace:** `welcomebot-prod`
- **Sync:** Manual (requires explicit approval)
- **Self-Heal:** Disabled
- **Prune:** Enabled

## Viewing Status

### Via ArgoCD UI

```bash
# Port forward
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Open browser: https://localhost:8080
# Login with admin credentials
```

### Via CLI

```bash
# Install ArgoCD CLI
brew install argocd  # macOS
# or download from: https://github.com/argoproj/argo-cd/releases

# Login
argocd login localhost:8080

# List applications
argocd app list

# Get application details
argocd app get welcomebot-staging

# View sync history
argocd app history welcomebot-staging

# View logs
argocd app logs welcomebot-staging
```

## Manual Operations

### Sync Now

```bash
# Sync immediately (don't wait for automatic sync)
argocd app sync welcomebot-staging

# Sync with dry-run
argocd app sync welcomebot-staging --dry-run

# Force sync
argocd app sync welcomebot-staging --force
```

### Rollback

```bash
# View history
argocd app history welcomebot-staging

# Rollback to specific revision
argocd app rollback welcomebot-staging 5

# Or use UI: History → Select revision → Rollback
```

### Refresh

```bash
# Refresh (check Git for changes)
argocd app refresh welcomebot-staging
```

## Troubleshooting

### Application stuck "Progressing"

```bash
# Check events
kubectl get events -n welcomebot-staging --sort-by='.lastTimestamp'

# Check pod status
kubectl get pods -n welcomebot-staging

# View ArgoCD app controller logs
kubectl logs -n argocd deployment/argocd-application-controller
```

### Sync failed

```bash
# Check sync status
argocd app get welcomebot-staging

# View detailed sync result
argocd app sync welcomebot-staging --dry-run --local deployments/overlays/staging

# Check if manifests are valid
kubectl apply -k deployments/overlays/staging --dry-run=client
```

### Out of sync

```bash
# Check what's different
argocd app diff welcomebot-staging

# Force sync
argocd app sync welcomebot-staging --force --replace
```

## Best Practices

1. **Always test locally first:**
   ```bash
   kubectl apply -k deployments/overlays/staging --dry-run=client
   ```

2. **Use Git tags for production**, branch commits for staging

3. **Monitor sync status** after pushing changes

4. **Use ArgoCD notifications** to alert on sync failures

5. **Set up RBAC** for team access control

## Security Notes

- ArgoCD has full cluster access (be careful who has ArgoCD access)
- Use separate service accounts for each application (future)
- Rotate admin password after initial setup
- Use SSO/OIDC for production (future)

## Links

- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [ArgoCD Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
- [Kustomize Documentation](https://kustomize.io/)

