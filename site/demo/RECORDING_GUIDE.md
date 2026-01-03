# Asciinema Recording Guide

This guide explains how to record real Asciinema demos from the executable scripts.

## Prerequisites

1. **Install Asciinema**
   ```bash
   # macOS
   brew install asciinema

   # Ubuntu/Debian
   apt-get install asciinema

   # Or use pip
   pip3 install asciinema
   ```

2. **Set up kind cluster**
   ```bash
   kind create cluster --name kspec-test --image kindest/node:v1.29.0
   ```

3. **Install dependencies**
   ```bash
   # Install cert-manager
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
   kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager

   # Install Kyverno v1.11.4
   kubectl create namespace kyverno
   kubectl apply --server-side=true -f https://github.com/kyverno/kyverno/releases/download/v1.11.4/install.yaml
   kubectl wait --for=condition=available --timeout=300s deployment/kyverno-admission-controller -n kyverno
   ```

4. **Build kspec**
   ```bash
   cd /path/to/kspec
   go build -o kspec ./cmd/kspec
   ```

## Recording Process

### 1. Scan Demo

```bash
# Set terminal size (important for consistent playback)
stty cols 120 rows 30

# Rename context to match prompt
kubectl config rename-context kind-kspec-test kind-kspec

# Record
cd site/demo/scripts
asciinema rec ../../public/demos/asciinema/scan.cast \
  --title "kspec scan demo" \
  --command "bash scan.sh" \
  --env="PS1=(kind-kspec) franck@csengineering$ "

# After recording completes, verify
asciinema play ../../public/demos/asciinema/scan.cast
```

### 2. Enforce Demo

```bash
cd site/demo/scripts
asciinema rec ../../public/demos/asciinema/enforce.cast \
  --title "kspec enforce demo" \
  --command "bash enforce.sh" \
  --env="PS1=(kind-kspec) franck@csengineering$ "
```

### 3. Drift Demo

```bash
# First, ensure policies are deployed
./kspec enforce --spec specs/examples/production.yaml

# Then record
cd site/demo/scripts
asciinema rec ../../public/demos/asciinema/drift.cast \
  --title "kspec drift demo" \
  --command "bash drift.sh" \
  --env="PS1=(kind-kspec) franck@csengineering$ "
```

### 4. Reports Demo

```bash
cd site/demo/scripts
asciinema rec ../../public/demos/asciinema/reports.cast \
  --title "kspec reports demo" \
  --command "bash reports.sh" \
  --env="PS1=(kind-kspec) franck@csengineering$ "
```

### 5. Metrics Demo

**Note:** Metrics demo requires kspec-operator to be deployed first.

```bash
# Deploy operator
kubectl create namespace kspec-system
kubectl apply -f deploy/manifests/

# Wait for operator
kubectl wait --for=condition=available --timeout=300s deployment/kspec-operator -n kspec-system

# Record
cd site/demo/scripts
asciinema rec ../../public/demos/asciinema/metrics.cast \
  --title "kspec metrics demo" \
  --command "bash metrics.sh" \
  --env="PS1=(kind-kspec) franck@csengineering$ "
```

## Post-Recording

1. **Verify recordings**
   ```bash
   for cast in public/demos/asciinema/*.cast; do
     echo "Checking $cast..."
     asciinema play "$cast"
   done
   ```

2. **Check file sizes**
   ```bash
   ls -lh public/demos/asciinema/*.cast
   ```

3. **Commit recordings**
   ```bash
   git add public/demos/asciinema/*.cast
   git commit -m "chore(site): update Asciinema recordings with real kspec output"
   ```

## Recording Tips

- **Timing**: Keep each demo between 45-90 seconds total
- **Pauses**: Add natural pauses with `sleep 1` between commands if needed
- **Errors**: If you make a mistake, press Ctrl+D to stop, then re-record
- **Prompt**: Ensure PS1 is set to `(kind-kspec) franck@csengineering$`
- **Terminal size**: Always use 120x30 (cols x rows)
- **Clean output**: Clear screen with `clear` before starting if needed

## Troubleshooting

**Problem**: Prompt doesn't show correctly
```bash
# Solution: Set PS1 explicitly
export PS1="(kind-kspec) franck@csengineering$ "
```

**Problem**: Recording cuts off output
```bash
# Solution: Increase terminal rows
stty rows 40
```

**Problem**: Context name is wrong
```bash
# Solution: Rename context
kubectl config rename-context $(kubectl config current-context) kind-kspec
```

## Automation (Future)

We plan to automate recording via CI:
```yaml
- name: Record demos
  run: |
    apt-get install -y asciinema
    ./scripts/record-all-demos.sh
```

For now, recordings are manually created and committed as artifacts.
