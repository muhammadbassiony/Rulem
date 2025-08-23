# Release Setup Guide for Rulem v0.1.0

This guide walks you through setting up automated releases for the Rulem CLI tool using GoReleaser, Homebrew, and Snap.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [GitHub Repository Setup](#github-repository-setup)
3. [Homebrew Tap Setup](#homebrew-tap-setup)
4. [Snap Store Setup](#snap-store-setup)
5. [Local Testing with Multipass](#local-testing-with-multipass)
6. [GitHub Secrets Configuration](#github-secrets-configuration)
7. [Release Process](#release-process)
8. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Accounts
- [x] GitHub account with repository access
- [ ] [Snapcraft.io account](https://snapcraft.io/) (Ubuntu One login)
- [ ] [Homebrew tap repository](https://github.com/new) (will create)

### Required Tools (macOS)
```bash
# Install Multipass for Ubuntu VM testing
brew install multipass

# Verify GoReleaser is working
goreleaser check
```

## GitHub Repository Setup

### 1. Update Configuration Files

Replace `yourusername` with your actual GitHub username in these files:
- `rulem/.goreleaser.yaml`
- `rulem/README.md`
- `rulem/.github/workflows/release.yml`

### 2. Verify GoReleaser Configuration

```bash
# Test local build
goreleaser release --snapshot --clean

# Check generated artifacts
ls -la dist/
```

## Homebrew Tap Setup

### 1. Create Homebrew Tap Repository

1. Go to [GitHub](https://github.com/new)
2. Create a new repository named `homebrew-rulem`
3. Initialize with README
4. Clone locally:

```bash
git clone https://github.com/yourusername/homebrew-rulem.git
cd homebrew-rulem
```

### 2. Update GoReleaser Configuration

Update `.goreleaser.yaml` with your GitHub username:

```yaml
# Add this section to .goreleaser.yaml
brews:
  - name: rulem
    homepage: "https://github.com/yourusername/rulem"
    description: "AI Assistant Instruction Manager CLI"
    license: "MIT"
    repository:
      owner: yourusername  # Replace with your username
      name: homebrew-rulem
```

### 3. Test Homebrew Integration

After your first release, users will install via:
```bash
brew tap yourusername/rulem
brew install rulem
```

## Snap Store Setup

### 1. Create Snapcraft Account

1. Visit [snapcraft.io](https://snapcraft.io/)
2. Click "Sign up" and use Ubuntu One account
3. Verify your email

### 2. Register Snap Name

**Important**: This must be done from a Linux environment (use Multipass VM):

```bash
# In Ubuntu VM (see Multipass section below)
snapcraft login
snapcraft register rulem
```

If "rulem" is taken, try variations:
- `rulem-cli`
- `ai-rulem`
- `rulem-manager`

Update `.goreleaser.yaml` if you use a different name:
```yaml
snapcrafts:
  - name: your-registered-name  # Update this
```

### 3. GoReleaser Snap Publishing

**GoReleaser WILL automatically publish to Snap Store** when configured with proper credentials. You don't need to manually publish.

## Local Testing with Multipass

### 1. Create Ubuntu VM

```bash
# Create VM with sufficient resources
multipass launch --name snap-test --cpus 2 --memory 4G --disk 20G 22.04

# Verify VM is running
multipass list

# Get VM info
multipass info snap-test
```

### 2. Mount Project Directory

```bash
# Mount your project (adjust path to your rulem directory)
multipass mount ~/path/to/rulem snap-test:/home/ubuntu/rulem

# Verify mount worked
multipass shell snap-test
ls -la /home/ubuntu/rulem/
```

### 3. Setup VM for Development

```bash
# Inside VM (multipass shell snap-test)
sudo apt update && sudo apt upgrade -y

# Install required tools
sudo apt install -y snapd snapcraft git curl

# Install GoReleaser
echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
sudo apt update
sudo apt install goreleaser

# Install Go
sudo snap install go --classic

# Verify installations
go version
goreleaser --version
snapcraft --version
```

### 4. Test Snap Build

```bash
# Inside VM, navigate to project
cd /home/ubuntu/rulem

# Build snapshot release
goreleaser release --snapshot --clean

# Check generated snap files
ls -la dist/*.snap

# Install snap locally for testing
sudo snap install --dangerous dist/rulem_*.snap

# Test the snap
rulem --version

# Test your TUI - use proper path for storage
# When prompted for storage directory, use:
# /home/ubuntu/rulem-storage  (absolute path within home)
# OR
# ~/rulem-storage             (relative to home)
rulem

# Check snap status
snap list | grep rulem
snap connections rulem
```

### 5. Debug Snap Issues

If you encounter permission issues:

```bash
# Check snap logs
sudo journalctl -u snapd --no-pager -l

# Try devmode for debugging (less secure but more permissive)
sudo snap remove rulem
sudo snap install --devmode --dangerous dist/rulem_*.snap

# Test again
rulem --version

# Check available interfaces
snap interface home
snap interface removable-media

# If still getting path errors, check home directory
echo "Home directory: $HOME"
echo "Trying path: /home/ubuntu/rulem-storage"

# Create storage directory if it doesn't exist
mkdir -p /home/ubuntu/rulem-storage
```

### 6. Get Snap Store Credentials

```bash
# Inside VM, login to Snap Store
snapcraft login
# Enter your Ubuntu One credentials

# Export credentials for automation
snapcraft export-login --snaps=rulem --channels=stable,candidate,beta,edge credentials.txt

# View credentials (copy this output)
cat credentials.txt
```

Copy the credentials file to your macOS:
```bash
# From macOS terminal
multipass transfer snap-test:/home/ubuntu/rulem/credentials.txt ~/Downloads/
```

### 7. VM Management Commands

```bash
# Stop VM to save resources
multipass stop snap-test

# Start VM
multipass start snap-test

# Restart VM
multipass restart snap-test

# Delete VM when completely done
multipass delete snap-test
multipass purge
```

## GitHub Secrets Configuration

### 1. Add Snap Store Credentials

1. Go to your GitHub repository
2. Navigate to: Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `SNAPCRAFT_STORE_CREDENTIALS`
5. Value: Paste the entire contents of `credentials.txt` from the VM

### 2. Verify GitHub Token

The `GITHUB_TOKEN` should be automatically available, but verify it has write permissions:
1. Go to Settings → Actions → General
2. Ensure "Workflow permissions" is set to "Read and write permissions"

### 3. Add Homebrew Token (if needed)

For private homebrew taps, you may need:
1. Create a Personal Access Token with `repo` scope
2. Add as secret: `HOMEBREW_TAP_GITHUB_TOKEN`

## Release Process

### 1. Pre-Release Checklist

- [ ] All code changes committed
- [ ] Tests passing locally
- [ ] GoReleaser config tested: `goreleaser check`
- [ ] Snap build tested in VM
- [ ] Version number decided (e.g., v0.1.0)

### 2. Create Release

```bash
# Ensure you're on main branch
git checkout main
git pull origin main

# Create and push tag
git tag v0.1.0
git push origin v0.1.0
```

### 3. Monitor Release

1. Watch GitHub Actions: `https://github.com/yourusername/rulem/actions`
2. Check GitHub Releases: `https://github.com/yourusername/rulem/releases`
3. Verify Snap Store: `https://snapcraft.io/rulem`
4. Check Homebrew tap: `https://github.com/yourusername/homebrew-rulem`

### 4. Post-Release Verification

```bash
# Test Homebrew installation
brew tap yourusername/rulem
brew install rulem
rulem --version

# Test Snap installation
sudo snap install rulem
rulem --version
```

## Troubleshooting

### GoReleaser Issues

```bash
# Check configuration
goreleaser check

# Test build without releasing
goreleaser release --snapshot --clean

# View detailed logs
goreleaser release --snapshot --clean --verbose
```

### Snap Issues

```bash
# Check snap build logs in VM
snapcraft --debug

# Check installed snap logs
snap logs rulem

# Check snap connections
snap connections rulem
```

### Homebrew Issues

- Verify `homebrew-rulem` repository exists and is public
- Check GitHub token permissions
- Ensure `.goreleaser.yaml` has correct repository owner/name

### Path Validation Issues in VM

If you get "path must be within your home directory" error when setting up storage:

```bash
# In VM: Check what your home directory is
echo $HOME
# Should show: /home/ubuntu

# Use a path within actual home directory, not mounted project
# WRONG: /home/ubuntu/rulem/devCentral (rulem is mounted from host)
# RIGHT: /home/ubuntu/rulem-storage

# Create and use a proper home directory path
mkdir -p /home/ubuntu/rulem-storage
# Then use "/home/ubuntu/rulem-storage" in setup

# Alternative: Use relative path from home
# In setup UI, enter: ~/rulem-storage
```

### VM Connection Issues

```bash
# If mount fails, use file transfer instead
multipass transfer ~/path/to/rulem/. snap-test:/home/ubuntu/rulem/

# If VM won't start
multipass restart snap-test

# If VM is corrupted
multipass delete snap-test
multipass launch --name snap-test 22.04
```

## Quick Reference Commands

### Daily Development Cycle
```bash
# Start VM and test
multipass start snap-test && multipass shell snap-test
cd /home/ubuntu/rulem
goreleaser release --snapshot --clean
sudo snap install --dangerous dist/rulem_*.snap && rulem --version

# Stop VM when done
exit && multipass stop snap-test
```

### Release Cycle
```bash
# Create release
git tag v0.1.0 && git push origin v0.1.0

# Monitor in GitHub Actions, then verify:
brew install yourusername/rulem/rulem
sudo snap install rulem
```

## Additional Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [Snapcraft Documentation](https://snapcraft.io/docs)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Multipass Documentation](https://multipass.run/docs)

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review GitHub Actions logs
3. Test locally with `goreleaser release --snapshot --clean`
4. Check snap logs with `snap logs rulem` in the VM
```

Now, let me help you debug the path validation issue. Looking at your setupmenu.go, the error suggests there's a validation that restricts paths to be within the home directory. Let me check what's happening: