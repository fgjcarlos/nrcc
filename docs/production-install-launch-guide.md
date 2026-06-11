# Production installation and launch guide

## Goal

Deploy the `nrcc` bootstrap installer so that `https://get.nrcc.dev/install.sh` is served via GitHub Pages and the following end-to-end flow works:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sh
sudo nrcc install
```

## Prerequisites

- Admin access to the `fgjcarlos/nrcc` repository.
- Access to GitHub Pages for the repository.
- Access to the `nrcc.dev` DNS provider.
- Confirm that the following files already exist on the published branch:
  - `docs/index.html`
  - `docs/install.sh`
  - `docs/CNAME`
  - `.github/workflows/release.yml`
- A Linux or macOS host available to validate the final installation.
- `curl` available on the test machine.

## GitHub Pages checklist

1. Go to `GitHub → fgjcarlos/nrcc → Settings → Pages`.
2. Under `Build and deployment`, select:
   - `Source`: `Deploy from a branch`
   - `Branch`: `main`
   - `Folder`: `/docs`
3. Save the configuration.
4. Confirm that GitHub detects the custom domain `get.nrcc.dev`.
5. Wait for Pages to publish the site.
6. Verify that the Pages URL responds before validating the final custom domain.

Notes:

- `docs/CNAME` must contain exactly `get.nrcc.dev`.
- If Pages shows domain warnings, they normally disappear once DNS has propagated correctly.

## DNS and CNAME checklist

1. Go to the `nrcc.dev` DNS control panel.
2. Create or update the record:

```dns
Type:        CNAME
Name:        get
Destination: fgjcarlos.github.io
```

3. Remove any conflicting records for `get.nrcc.dev` (duplicate `A`, `AAAA`, or `CNAME` records).
4. Wait for DNS propagation.
5. Verify resolution:

```bash
dig +short get.nrcc.dev
dig +short CNAME get.nrcc.dev
```

Expected result:

- `dig +short CNAME get.nrcc.dev` should return `fgjcarlos.github.io.` or equivalent.
- Full resolution may take several minutes depending on TTL and DNS provider.

## Release / test tag checklist

The `.github/workflows/release.yml` workflow runs on push of `v*` tags.

1. Choose a test version, for example `v0.1.0` or the next available version.
2. Create the annotated tag locally:

```bash
git tag -a v0.1.0 -m "Test release v0.1.0"
```

3. Push the tag:

```bash
git push origin v0.1.0
```

4. Go to `GitHub → Actions` and confirm the `Release` workflow completes successfully.
5. Go to `GitHub → Releases` and verify that the release exists for that tag.
6. Confirm the release attaches at least:
   - `nrcc-linux-amd64`
   - `nrcc-linux-arm64`
   - `nrcc-linux-armv7`
   - `nrcc-darwin-amd64`
   - `nrcc-darwin-arm64`
   - `nrcc-windows-amd64.exe`
   - `SHA256SUMS`
   - `*.sha256`

## Installer and `nrcc install` validation checklist

### Remote bootstrap validation

1. Confirm the published script responds:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sed -n '1,20p'
```

2. Validate installation on a clean or controlled machine:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sh
```

3. Confirm the binary was installed:

```bash
which nrcc
nrcc --version
```

4. Validate pinned version:

```bash
NRCC_VERSION=v0.1.0 curl -fsSL https://get.nrcc.dev/install.sh | sh
nrcc --version
```

### `nrcc install` validation

1. Run:

```bash
sudo nrcc install
```

2. Confirm the service is installed and started.

   On a clean Linux VM, validate each Node-RED mode:
   - `native`: if `node` or `npm` are missing, the installer must install them before running `npm install -g node-red`.
   - `docker`: if `docker` is missing, the installer must install Docker and verify `docker info` before creating the Node-RED container.
   - `skip`: must not install Node-RED or any Node-RED-specific dependencies.
   - `--with-portless`: if Portless is enabled and `node`/`npm` are missing, must set up Node.js/npm before installing Portless.

3. Verify status:

```bash
systemctl status nrcc
```

4. Open `http://localhost:3001`.
5. Complete the initial admin user setup.

If you are not using `systemd`, verify at minimum that the downloaded binary starts and that the next step shown by the installer is consistent with your target platform.

## Verifying that `get.nrcc.dev` works

Use these checks in order:

1. DNS:

```bash
dig +short CNAME get.nrcc.dev
```

2. HTTP response from the site:

```bash
curl -I https://get.nrcc.dev/
```

3. Installer response:

```bash
curl -I https://get.nrcc.dev/install.sh
```

4. Expected installer content:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | grep 'REPO="fgjcarlos/nrcc"'
```

5. End-to-end flow:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sh
sudo nrcc install
```

Expected results:

- `https://get.nrcc.dev/` serves the installation landing page.
- `https://get.nrcc.dev/install.sh` returns the correct shell script.
- The installer resolves the latest release from GitHub and downloads the matching binary.

## Basic troubleshooting

### GitHub Pages is not publishing

- Check `Settings → Pages` and confirm `main + /docs`.
- Confirm that `docs/index.html` exists on the published branch.
- Confirm that `docs/CNAME` contains exactly `get.nrcc.dev`.
- Wait a few minutes and retry; Pages does not always update immediately.

### `get.nrcc.dev` does not resolve or resolves incorrectly

- Verify the record is `CNAME get → fgjcarlos.github.io`.
- Remove duplicate or conflicting records for the `get` subdomain.
- Check propagation with `dig` from more than one network if needed.

### `curl https://get.nrcc.dev/install.sh` returns 404

- Confirm that GitHub Pages is publishing the `/docs` folder.
- Confirm the file is named exactly `docs/install.sh`.
- Confirm the Pages deployment has completed.

### Installer cannot find the latest version

- Verify that a published GitHub Release exists.
- Verify the tag follows the `v*` pattern to trigger the workflow.
- Test with `NRCC_VERSION=<tag>` to isolate whether the issue is with `latest` resolution.

### Installer fails checksum verification

- Verify the release includes `*.sha256` and `SHA256SUMS` files.
- Confirm the `.sha256` sidecars point to the normalised `nrcc` filename expected by the script.

### `sudo nrcc install` fails

- Check permissions, host dependencies, and whether `systemd` is available.
- Run `nrcc --help` and `nrcc install --help` to review available flags.
- Manually verify the downloaded binary runs before attempting service installation.

## Next steps after validation

1. Create the real launch release if the test was successful.
2. Update the example version in documentation if needed.
3. Add validation of `get.nrcc.dev/install.sh` to the operational release checklist.
4. Consider an automated post-release check to verify:
   - `https://get.nrcc.dev/`
   - `https://get.nrcc.dev/install.sh`
   - installation with `NRCC_VERSION=<tag>`
5. Officially announce the one-liner flow once the domain and release are stable.
