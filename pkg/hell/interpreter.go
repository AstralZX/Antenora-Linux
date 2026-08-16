package hell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Interpreter executes a parsed Hell package's build, install and post_install
// instruction sequences inside a build environment.
type Interpreter struct {
	Env             map[string]string
	SrcDir          string
	Root            string
	Arch            string
	BinaryAvailable bool
	Verbose         bool
	DryRun          bool
	Out             io.Writer
}

// NewInterpreter constructs an Interpreter with an empty environment and the
// given staging root, source directory and target architecture.
func NewInterpreter(root, srcDir, arch string) *Interpreter {
	return &Interpreter{
		Env:    make(map[string]string),
		Root:   root,
		SrcDir: srcDir,
		Arch:   arch,
		Out:    os.Stdout,
	}
}

// Setup seeds the predefined Hell variables, honouring architecture-specific
// flags when present.
func (in *Interpreter) Setup(pkg *Package, jobs string) {
	in.Env["HELL_ROOT"] = in.Root
	in.Env["HELL_SRC_DIR"] = in.SrcDir
	in.Env["HELL_ARCH"] = in.Arch
	in.Env["HELL_CFLAGS"] = pkg.CFlags
	in.Env["HELL_LDFLAGS"] = pkg.LDFlags
	if jobs == "" {
		jobs = NumJobs()
	}
	in.Env["HELL_JOBS"] = jobs
	in.Env["MAKEOPTS"] = "-j" + jobs
	if a, ok := pkg.Arches[in.Arch]; ok {
		if a.CFlags != "" {
			in.Env["HELL_CFLAGS"] = a.CFlags
		}
		if a.LDFlags != "" {
			in.Env["HELL_LDFLAGS"] = a.LDFlags
		}
	}
}

// Exec runs a sequence of statements, short-circuiting on the first error.
func (in *Interpreter) Exec(stmts []Stmt) error {
	for _, s := range stmts {
		if err := in.execStmt(s); err != nil {
			return fmt.Errorf("hell: line %d: %w", s.Line, err)
		}
	}
	return nil
}

func (in *Interpreter) execStmt(s Stmt) error {
	switch s.Op {
	case "if":
		if in.evalCond(s.Cond, s.CondArg) {
			return in.Exec(s.Then)
		}
		return in.Exec(s.Else)
	case "run":
		cmd := expandEnv(s.Args[0], in.Env)
		return runShell(in.SrcDir, in.Env, cmd, in.DryRun, in.Verbose, in.Out)
	case "patch":
		patchFile := expandEnv(s.Args[1], in.Env)
		if !filepath.IsAbs(patchFile) {
			patchFile = filepath.Join(in.SrcDir, patchFile)
		}
		if in.DryRun {
			fmt.Fprintf(in.Out, "  + patch -p1 < %s\n", patchFile)
			return nil
		}
		return applyPatch(in.SrcDir, patchFile)
	case "mkdir":
		path := in.rooted(expandEnv(s.Args[0], in.Env))
		if in.DryRun {
			fmt.Fprintf(in.Out, "  + mkdir -p %s\n", path)
			return nil
		}
		return os.MkdirAll(path, 0o755)
	case "cp":
		src := in.rooted(expandEnv(s.Args[0], in.Env))
		dst := in.rooted(expandEnv(s.Args[1], in.Env))
		if in.DryRun {
			fmt.Fprintf(in.Out, "  + cp -a %s %s\n", src, dst)
			return nil
		}
		return copyPath(src, dst)
	case "rm":
		path := in.rooted(expandEnv(s.Args[0], in.Env))
		if in.DryRun {
			fmt.Fprintf(in.Out, "  + rm -rf %s\n", path)
			return nil
		}
		return os.RemoveAll(path)
	case "ln":
		target := expandEnv(s.Args[0], in.Env)
		link := in.rooted(expandEnv(s.Args[1], in.Env))
		if in.DryRun {
			fmt.Fprintf(in.Out, "  + ln -s %s %s\n", target, link)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			return err
		}
		return os.Symlink(target, link)
	case "var":
		in.Env[s.Args[0]] = expandEnv(s.Args[1], in.Env)
		return nil
	}
	return fmt.Errorf("unknown instruction %q", s.Op)
}

// rooted maps an absolute path into the staging root so that install steps
// using absolute paths do not touch the host filesystem.
func (in *Interpreter) rooted(p string) string {
	if in.Root == "" || in.Root == "/" {
		return p
	}
	if filepath.IsAbs(p) {
		return filepath.Join(in.Root, p)
	}
	return p
}

func (in *Interpreter) evalCond(cond, arg string) bool {
	switch cond {
	case "arch_x86_64":
		return in.Arch == "x86_64"
	case "arch_aarch64":
		return in.Arch == "aarch64"
	case "binary_available":
		return in.BinaryAvailable
	case "file_exists":
		p := expandEnv(arg, in.Env)
		if !filepath.IsAbs(p) {
			p = filepath.Join(in.SrcDir, p)
		}
		_, err := os.Stat(p)
		return err == nil
	case "env":
		_, ok := in.Env[arg]
		return ok
	}
	return false
}
