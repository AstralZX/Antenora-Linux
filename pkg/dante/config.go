package dante

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// Config holds the Dante configuration parsed from /etc/dante/dante.conf.
// Defaults are applied for any key that is absent.
type Config struct {
	Binary       string
	MakeFlags    string
	RepoURL      string
	BinaryMirror string
	CleanSource  string
	KeepDeps     string
	RepoDir      string
	DBDir        string
	CacheDir     string
	BuildDir     string
	Root         string
	GPGKeyID     string
}

// DefaultConfig returns the canonical Antenora defaults. RepoDir, DBDir and
// BuildDir can be overridden via the DANTE_ROOT environment variable so the
// manager is testable and bootstrapable inside a chroot or container.
func DefaultConfig() *Config {
	prefix := os.Getenv("DANTE_ROOT")
	if prefix == "" {
		prefix = ""
	}
	c := &Config{
		Binary:       "NO",
		MakeFlags:    "-j$(nproc)",
		RepoURL:      "https://github.com/antenora/package-repo.git",
		BinaryMirror: "https://cdn.antenora.org/packages",
		CleanSource:  "YES",
		KeepDeps:     "NO",
		RepoDir:      filepath.Join(prefix, "var/lib/dante/repo"),
		DBDir:        filepath.Join(prefix, "var/lib/dante/db"),
		CacheDir:     filepath.Join(prefix, "var/cache/dante"),
		BuildDir:     filepath.Join(prefix, "var/tmp/dante/build"),
		Root:         "/",
		GPGKeyID:     "0x4E54454E4F5241",
	}
	if prefix != "" {
		c.Root = prefix
	}
	return c
}

// LoadConfig reads the configuration file at path, falling back to defaults
// when the file does not exist. It also honours environment variables that
// override any value (e.g. DANTE_BINARY=YES).
func LoadConfig(path string) (*Config, error) {
	c := DefaultConfig()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return applyEnvOverrides(c), nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "BINARY":
			c.Binary = val
		case "MAKEFLAGS":
			c.MakeFlags = val
		case "REPO_URL":
			c.RepoURL = val
		case "BINARY_MIRROR":
			c.BinaryMirror = val
		case "CLEAN_SOURCE":
			c.CleanSource = val
		case "KEEP_DEPS":
			c.KeepDeps = val
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return applyEnvOverrides(c), nil
}

func applyEnvOverrides(c *Config) *Config {
	override(&c.Binary, "DANTE_BINARY")
	override(&c.MakeFlags, "DANTE_MAKEFLAGS")
	override(&c.RepoURL, "DANTE_REPO_URL")
	override(&c.BinaryMirror, "DANTE_BINARY_MIRROR")
	if v, ok := os.LookupEnv("DANTE_GPG_KEY"); ok {
		c.GPGKeyID = v
	}
	return c
}

func override(dst *string, key string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

// BinaryEnabled reports whether binary packages are preferred.
func (c *Config) BinaryEnabled() bool {
	return strings.EqualFold(c.Binary, "YES")
}

// CleanSourceEnabled reports whether source trees are removed after a build.
func (c *Config) CleanSourceEnabled() bool {
	return !strings.EqualFold(c.CleanSource, "NO")
}

// KeepDepsEnabled reports whether dependencies are retained on removal.
func (c *Config) KeepDepsEnabled() bool {
	return strings.EqualFold(c.KeepDeps, "YES")
}

// Jobs extracts the -jN value from MakeFlags, defaulting to the CPU count.
func (c *Config) Jobs() string {
	f := strings.Fields(c.MakeFlags)
	for _, flag := range f {
		if strings.HasPrefix(flag, "-j") {
			v := strings.TrimPrefix(flag, "-j")
			if v == "" {
				return fmt.Sprintf("%d", runtime.NumCPU())
			}
			return v
		}
	}
	return fmt.Sprintf("%d", runtime.NumCPU())
}

// EnsureDirs creates the directories Dante needs to operate.
func (c *Config) EnsureDirs() error {
	for _, d := range []string{c.RepoDir, c.DBDir, c.CacheDir, c.BuildDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// RequireRoot verifies the manager is running with sufficient privilege.
func RequireRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("dante: this operation requires root privileges")
	}
	return nil
}

// CurrentUser returns the username for the sudo/regular user, used by
// post_install steps that need to place files in a home directory.
func CurrentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "root" {
		return u.Username
	}
	if v := os.Getenv("SUDO_USER"); v != "" {
		return v
	}
	if v := os.Getenv("DANTE_USER"); v != "" {
		return v
	}
	return ""
}
