// Package selfupdate lets a running pipe binary replace itself with the
// latest GitHub release. It is exposed through the --update and
// --update-check CLI flags.
package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/MachuraHarry/pipe/pkg/build"
)

const Repo = "MachuraHarry/pipe"

// Overridable in tests to serve fake releases from a local server.
var (
	latestReleaseURL = "https://api.github.com/repos/" + Repo + "/releases/latest"
	downloadBase     = "https://github.com/" + Repo + "/releases/download"
)

func ReleaseURL(tag string) string {
	return "https://github.com/" + Repo + "/releases/tag/" + tag
}

func BinaryName(goos string) string {
	if goos == "windows" {
		return "pipe.exe"
	}
	return "pipe"
}

func ArtifactName(goos, goarch string) string {
	return fmt.Sprintf("pipe-%s-%s.tar.gz", goos, goarch)
}

// SupportedPlatform mirrors the release build matrix in
// .github/workflows/release.yml: linux/darwin on amd64+arm64, Windows on
// amd64 only.
func SupportedPlatform(goos, goarch string) bool {
	switch goos + "/" + goarch {
	case "linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64":
		return true
	default:
		return false
	}
}

type ghRelease struct {
	TagName string `json:"tag_name"`
}

// LatestRelease returns the tag of the newest published GitHub release.
func LatestRelease() (string, error) {
	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach GitHub (offline?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return "", fmt.Errorf("GitHub API rate limit hit (%s); retry later", resp.Status)
		}
		return "", fmt.Errorf("GitHub API error: %s", resp.Status)
	}
	var r ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decoding release info: %w", err)
	}
	if r.TagName == "" {
		return "", fmt.Errorf("GitHub API returned no tag name")
	}
	return r.TagName, nil
}

func parseSemver(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || parts[0] == "" {
		return out, false
	}
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 || len(parts[i]) > 9 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// UpdateNeeded reports whether latest is newer than current. Versions built
// with `git describe` (e.g. v1.2.3-4-gabc1234 or v1.2.3-dirty) compare by
// their base semver, so they do not nag when already on the latest tag.
// Unparsable versions only trigger an update when they differ from latest.
func UpdateNeeded(current, latest string) bool {
	if strings.TrimPrefix(current, "v") == strings.TrimPrefix(latest, "v") {
		return false
	}
	cur, okCur := parseSemver(current)
	lat, okLat := parseSemver(latest)
	if !okLat {
		return false
	}
	if !okCur {
		return true
	}
	for i := range cur {
		if lat[i] != cur[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

// Run checks for a newer release and, unless checkOnly is set, replaces the
// running binary with it.
func Run(version string, checkOnly bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating executable: %w", err)
	}
	if resolved, lerr := filepath.EvalSymlinks(exePath); lerr == nil {
		exePath = resolved
	}
	return runUpdate(version, checkOnly, exePath)
}

func runUpdate(version string, checkOnly bool, exePath string) error {
	fmt.Println("Checking for updates ...")
	latest, err := LatestRelease()
	if err != nil {
		return err
	}

	if !UpdateNeeded(version, latest) {
		fmt.Printf("Pipe %s is up to date (latest release: %s)\n", version, latest)
		return nil
	}

	if checkOnly {
		fmt.Printf("Update available: %s -> %s\n", version, latest)
		fmt.Printf("Run 'pipe --update' to install it.\n%s\n", ReleaseURL(latest))
		return nil
	}

	goos, goarch := runtime.GOOS, runtime.GOARCH
	if !SupportedPlatform(goos, goarch) {
		return fmt.Errorf("no prebuilt binary for %s/%s; build from source instead", goos, goarch)
	}

	if _, embedded := build.LoadEmbedded(exePath); embedded && os.Getenv("PIPE_UPDATE_EMBEDDED") == "" {
		return fmt.Errorf("this binary embeds a Pipe program (-build) which an update would discard; set PIPE_UPDATE_EMBEDDED=1 to proceed anyway")
	}

	tmp, err := os.MkdirTemp("", "pipe-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	base := downloadBase + "/" + latest
	artifact := ArtifactName(goos, goarch)

	fmt.Printf("Downloading Pipe %s (%s/%s) ...\n", latest, goos, goarch)
	tarball := filepath.Join(tmp, artifact)
	if err := download(base+"/"+artifact, tarball); err != nil {
		return err
	}

	if err := download(base+"/"+artifact+".sha256", tarball+".sha256"); err == nil {
		if err := verifySHA256(tarball, tarball+".sha256"); err != nil {
			return err
		}
		fmt.Println("SHA256 verified")
	} else {
		fmt.Println("warning: no SHA256 checksum available, skipping verification")
	}

	binaryPath, err := extract(tarball, tmp, goos)
	if err != nil {
		return err
	}

	if err := replaceBinary(exePath, binaryPath); err != nil {
		return err
	}

	fmt.Printf("Updated: %s -> %s\n", version, latest)
	fmt.Println(ReleaseURL(latest))
	fmt.Printf("%s -h to verify; restart any running REPL.\n", exePath)
	return nil
}

func download(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func verifySHA256(path, sumFile string) error {
	data, err := os.ReadFile(sumFile)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return fmt.Errorf("empty checksum file")
	}
	want := strings.ToLower(fields[0])
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("SHA256 verification failed (expected %s, got %s)", want, got)
	}
	return nil
}

func extract(tarball, dir, goos string) (string, error) {
	f, err := os.Open(tarball)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("%s is not valid gzip: %w", filepath.Base(tarball), err)
	}
	defer gz.Close()

	name := BinaryName(goos)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if strings.Contains(hdr.Name, "..") || filepath.Base(hdr.Name) != name {
			continue
		}
		out := filepath.Join(dir, name)
		of, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(of, tr); err != nil {
			of.Close()
			return "", err
		}
		of.Close()
		return out, nil
	}
	return "", fmt.Errorf("archive did not contain %s", name)
}

// replaceBinary atomically moves newBinary over target. On Windows the
// running executable must be renamed away before the new one can take its
// place.
func replaceBinary(target, newBinary string) error {
	data, err := os.ReadFile(newBinary)
	if err != nil {
		return err
	}
	staged := target + ".new"
	if err := os.WriteFile(staged, data, 0755); err != nil {
		return fmt.Errorf("writing %s: %w (no permission? install to a user-writable PIPE_DIR)", staged, err)
	}
	if runtime.GOOS == "windows" {
		backup := target + ".old"
		os.Remove(backup)
		if err := os.Rename(target, backup); err != nil {
			return err
		}
		if err := os.Rename(staged, target); err != nil {
			os.Rename(backup, target)
			return err
		}
		os.Remove(backup)
		return nil
	}
	return os.Rename(staged, target)
}
