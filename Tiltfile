version_settings(constraint=">=0.22.2")
secret_settings(disable_scrub=True)
load('ext://dotenv', 'dotenv')

# Load tilt env file if it exists
dotenv_path = ".tilt.env"
if os.path.exists(dotenv_path):
  dotenv(fn=dotenv_path)

# Configure trigger mode
true = ("true", "1", "yes", "t", "y")
trigger_mode = TRIGGER_MODE_MANUAL
if os.environ.get('TRIGGER_MODE_AUTO', '').lower() in true:
  trigger_mode = TRIGGER_MODE_AUTO

# Docker images
custom_build(
  ref="preprocessing-sfa-worker:dev",
  command=["hack/build_docker.sh"],
  deps=["."],
)
docker_build(ref="apis-mock:dev", context="hack/apis-mock")

# Kubernetes manifests
k8s_yaml(kustomize("hack/kube"))

# Tilt resources
k8s_resource(
  "preprocessing-worker",
  labels=["SFA"],
  trigger_mode=trigger_mode,
)
k8s_resource(
  "apis-mock",
  port_forwards="8081:8080",
  labels=["SFA"],
  trigger_mode=trigger_mode,
)
k8s_resource("mysql-create-prep-database", labels=["SFA"])
k8s_resource(
  "mysql-recreate-prep-database",
  labels=["SFA"],
  auto_init=False,
  trigger_mode=TRIGGER_MODE_MANUAL,
)
