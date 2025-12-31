# Release Process for v0.3.1

This document provides exact commands for completing the v0.3.1 release.

## Prerequisites

Ensure you have:
- [ ] All tests passing (`go test ./...` = 134 tests PASS)
- [ ] All changes committed on `release/v0.3.1` branch
- [ ] Git configured with your name and email
- [ ] GitHub CLI (`gh`) installed and authenticated
- [ ] Push access to the repository

## Step 1: Verify Current State

```bash
# Confirm you're on the release branch
git branch --show-current
# Expected: release/v0.3.1

# Verify clean working tree
git status
# Expected: "nothing to commit, working tree clean"

# Verify all tests pass
go test ./...
# Expected: All packages PASS (134 tests)

# View commits since v0.3.0
git log v0.3.0..HEAD --oneline
# Expected: Release preparation commits visible
```

## Step 2: Merge Release Branch to Main

```bash
# Switch to main branch
git checkout main

# Pull latest changes (if any)
git pull origin main

# Merge release branch (no fast-forward to preserve history)
git merge --no-ff release/v0.3.1 -m "Merge release/v0.3.1: kspec v0.3.1 patch release

This patch release includes Phase 9 Track 1 testing and quality improvements:
- 36 new tests added (drift, metrics, advanced policies)
- Test coverage improved from ~50-60% to ~65-70%
- Fixed all 8 previously skipped drift tests
- No breaking changes - backward compatible with v0.3.0

Release highlights:
- Zero skipped tests
- Comprehensive metrics validation (19 tests)
- Advanced policy feature tests (9 tests)
- Production readiness improvements
- E2E smoke test automation

See RELEASE_NOTES_v0.3.1.md for complete details."

# Verify merge was successful
git log --oneline -5
```

## Step 3: Create Annotated Git Tag

```bash
# Create annotated tag v0.3.1 on main branch
git tag -a v0.3.1 -m "kspec v0.3.1 - Testing & Quality Improvements

Release Date: 2025-12-30
Type: Patch Release

Highlights:
- Zero skipped tests (fixed all 8 drift tests)
- 36 new tests added across 3 packages
- Test coverage: ~65-70% (up from ~50-60%)
- CI/CD stability improvements
- Production hardening (Phase 9 Track 1)

Breaking Changes: None
Upgrade: Drop-in replacement for v0.3.0

Full release notes: RELEASE_NOTES_v0.3.1.md
Changelog: CHANGELOG.md#031---2025-12-30"

# Verify tag was created
git tag -l -n9 v0.3.1

# Show tag details
git show v0.3.1 --stat
```

## Step 4: Push to Remote

```bash
# Push main branch with the merge commit
git push origin main

# Push the v0.3.1 tag
git push origin v0.3.1

# Verify tag is on remote
git ls-remote --tags origin | grep v0.3.1
```

## Step 5: Create GitHub Release

```bash
# Create GitHub Release using gh CLI
gh release create v0.3.1 \
  --title "kspec v0.3.1 - Testing & Quality Improvements" \
  --notes-file RELEASE_NOTES_v0.3.1.md \
  --target main

# Alternative: If you want to manually edit the release notes
gh release create v0.3.1 \
  --title "kspec v0.3.1 - Testing & Quality Improvements" \
  --notes "$(cat RELEASE_NOTES_v0.3.1.md)" \
  --target main \
  --draft  # Remove --draft when ready to publish

# Verify release was created
gh release view v0.3.1
```

## Step 6: Trigger Container Image Build

**If using GitHub Actions:**

The git tag push should automatically trigger the image build workflow.

```bash
# Check workflow run status
gh run list --workflow=docker-build.yml --limit 5

# Watch the workflow (if running)
gh run watch
```

**Manual image build (if needed):**

```bash
# Build the container image
make docker-build IMG=ghcr.io/cloudcwfranck/kspec-operator:v0.3.1

# Login to GHCR (if not already)
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USERNAME --password-stdin

# Push the image
docker push ghcr.io/cloudcwfranck/kspec-operator:v0.3.1

# Tag as latest (optional - only if this is the latest stable release)
docker tag ghcr.io/cloudcwfranck/kspec-operator:v0.3.1 ghcr.io/cloudcwfranck/kspec-operator:latest
docker push ghcr.io/cloudcwfranck/kspec-operator:latest
```

## Step 7: Verification Checklist

After release, verify:

```bash
# 1. Tag exists on GitHub
gh release view v0.3.1

# 2. Container image is available
docker pull ghcr.io/cloudcwfranck/kspec-operator:v0.3.1

# 3. Release notes are visible
gh release view v0.3.1 --web

# 4. Changelog is updated on main
git show main:CHANGELOG.md | head -50 | grep "0.3.1"

# 5. Run E2E smoke test (optional but recommended)
./scripts/e2e-v0.3.1-kind.sh
```

## Step 8: Post-Release Cleanup

```bash
# Optionally delete the release branch (if no longer needed)
git branch -d release/v0.3.1
git push origin --delete release/v0.3.1

# Create an issue for next release planning (optional)
gh issue create \
  --title "Plan v0.3.2 release (Phase 9 Track 2-6)" \
  --body "Next release should include:
- Alert integrations (Slack, PagerDuty, webhooks)
- Technical debt resolution (18 TODOs)
- Security hardening (image signing, SBOM)
- OpenTelemetry distributed tracing
- Comprehensive documentation

See docs/PHASE_9_PLAN.md for full roadmap." \
  --label "release-planning"
```

## Rollback Procedure (if needed)

If critical issues are discovered after release:

```bash
# 1. Delete the GitHub release
gh release delete v0.3.1 --yes

# 2. Delete the tag locally and remotely
git tag -d v0.3.1
git push origin --delete v0.3.1

# 3. Revert the merge commit on main (if needed)
git checkout main
git revert -m 1 HEAD  # If the merge was the last commit
git push origin main

# 4. Fix issues on a new branch
git checkout -b hotfix/v0.3.1-fix
# Make fixes, commit, test
git push origin hotfix/v0.3.1-fix

# 5. Re-run release process with v0.3.1 or v0.3.2
```

## Summary of Files Changed

All changes committed on `release/v0.3.1` branch:

### Modified Files:
1. **config/manager/manager.yaml**
   - Updated version labels: v0.2.0 → v0.3.1 (lines 9, 26)
   - Updated image tag: `:latest` → `:v0.3.1` (line 64)

2. **config/crd/kspec.io_clusterspecifications.yaml**
   - Regenerated with controller-gen v0.16.5
   - Added Phase 7 advanced policy fields (+180 lines)

3. **CHANGELOG.md**
   - Added v0.3.1 section with categorized entries

### New Files:
4. **RELEASE_NOTES_v0.3.1.md**
   - Comprehensive release documentation (214 lines)

5. **scripts/e2e-v0.3.1-kind.sh**
   - E2E smoke test automation (executable)

### Test Files (committed earlier on main):
6. **pkg/drift/testhelpers_test.go** - Kyverno GVR registration helpers
7. **pkg/metrics/metrics_test.go** - 19 metrics tests
8. **pkg/policy/advanced_test.go** - 9 advanced policy tests

## Contact

For release questions or issues:
- File an issue: https://github.com/cloudcwfranck/kspec/issues
- Review Phase 9 plan: docs/PHASE_9_PLAN.md

---

**Release Manager:** Claude Code (Automated Release Engineering)
**Release Date:** 2025-12-30
**Next Release:** v0.3.2 (Phase 9 Track 2-6)
