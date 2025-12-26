# kspec Installation Scripts

## Quick Install (Recommended)

### One-Line Installation

```bash
curl -sSL https://raw.githubusercontent.com/cloudcwfranck/kspec/main/scripts/install.sh | bash
```

This will:
- Auto-detect your operating system and architecture
- Download the latest kspec binary
- Install it to `/usr/local/bin`
- Verify the installation

### Custom Installation Directory

```bash
export INSTALL_DIR="$HOME/.local/bin"
curl -sSL https://raw.githubusercontent.com/cloudcwfranck/kspec/main/scripts/install.sh | bash
```

### Specific Version

```bash
export VERSION="v0.1.0"
curl -sSL https://raw.githubusercontent.com/cloudcwfranck/kspec/main/scripts/install.sh | bash
```

## After Installation

Once kspec is installed, run the interactive setup wizard:

```bash
kspec init
```

This will guide you through:
1. Detecting your Kubernetes cluster
2. Configuring security requirements
3. Generating a cluster specification
4. Optionally enforcing policies
5. Optionally setting up drift monitoring

## Manual Installation

If you prefer to install manually:

1. **Download the binary for your platform:**
   - Go to https://github.com/cloudcwfranck/kspec/releases/latest
   - Download the appropriate archive for your OS/architecture

2. **Extract and install:**
   ```bash
   tar -xzf kspec_*.tar.gz  # or unzip for Windows
   sudo mv kspec /usr/local/bin/
   chmod +x /usr/local/bin/kspec
   ```

3. **Verify:**
   ```bash
   kspec version
   ```

## Supported Platforms

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, arm64

## Troubleshooting

### "command not found: kspec"

The installation directory is not in your PATH. Add it:

```bash
export PATH="$PATH:/usr/local/bin"
```

Make it permanent by adding to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.).

### Permission Denied

The installer tries to use `sudo` when needed. If you don't have sudo access:

```bash
INSTALL_DIR="$HOME/.local/bin" curl -sSL https://raw.githubusercontent.com/cloudcwfranck/kspec/main/scripts/install.sh | bash
```

Then add `$HOME/.local/bin` to your PATH.

### SSL Certificate Issues

If you encounter SSL/TLS errors:

```bash
curl -sSL --insecure https://raw.githubusercontent.com/cloudcwfranck/kspec/main/scripts/install.sh | bash
```

**Note:** Only use `--insecure` if absolutely necessary and you trust the source.

## Uninstall

To remove kspec:

```bash
sudo rm /usr/local/bin/kspec
```

Or if installed to a custom directory:

```bash
rm $INSTALL_DIR/kspec
```
