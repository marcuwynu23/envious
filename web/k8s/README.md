# Apply order: namespace, config, secrets, storage, workload.
#   kubectl apply -f web/k8s/namespace.yaml
#   kubectl apply -f web/k8s/configmap.yaml
#   # create envious-secrets from the template in secret.yaml first!
#   kubectl apply -f web/k8s/pvc.yaml          # sqlite only; skip for postgres
#   kubectl apply -f web/k8s/deployment.yaml
#   kubectl apply -f web/k8s/service.yaml
#
# Postgres (managed DB or the compose db service as reference):
#   1. Set DATABASE_URL in envious-secrets and DB_DRIVER=postgres in
#      envious-config.
#   2. Delete pvc.yaml from your apply set and remove the data volume +
#      volumeMount from deployment.yaml.
#   3. Raise replicas and switch strategy to RollingUpdate.
#
# First-run API key: the server prints it once to stdout.
#   kubectl -n envious logs deploy/envious-web | grep 'initial API key'
