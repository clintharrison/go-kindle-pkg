package install

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/clintharrison/go-kindle-pkg/pkg/cli/clicommon"
	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/resolver"
	"github.com/clintharrison/go-kindle-pkg/pkg/state"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func repoInstalledMapToResolverVPkgMap(
	installed map[string][]*repository.RepoPackage,
) map[resolver.ArtifactID][]*resolver.VersionedPackage {
	vps := make(map[resolver.ArtifactID][]*resolver.VersionedPackage, len(installed))
	for pid, ps := range installed {
		for _, p := range ps {
			var cs []*resolver.Constraint
			for _, d := range p.Dependencies {
				cs = append(cs, &resolver.Constraint{
					ID:           resolver.ArtifactID(d.ID),
					Min:          d.Min,
					Max:          d.Max,
					RepositoryID: (*resolver.RepositoryID)(d.RepositoryID),
				})
			}
			vp := &resolver.VersionedPackage{
				ID:            resolver.ArtifactID(pid),
				RepositoryID:  resolver.RepositoryID(p.RepositoryID),
				Version:       p.Version,
				Dependencies:  cs,
				SupportedArch: p.SupportedArch,
			}

			vps[resolver.ArtifactID(pid)] = append(vps[resolver.ArtifactID(pid)], vp)
		}
	}
	return vps
}

func NewUninstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall [flags] package-id",
		Short: "Uninstall an installed package by its ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return errors.Wrap(err, "failed to get dry-run flag")
			}

			if len(args) < 1 {
				_ = cmd.Usage()
				_, _ = cmd.OutOrStderr().Write([]byte("\n"))
				return nil
			}
			packageID := args[0]

			installedPackages, err := state.GetInstalledPackages()
			if err != nil {
				return errors.Wrap(err, "failed to get installed packages")
			}
			uninstallPkgs, ok := installedPackages[packageID]
			if !ok || len(uninstallPkgs) == 0 {
				return fmt.Errorf("package %q is not installed", packageID)
			}

			multirepo, err := clicommon.GetRepoFromArgs(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to initialize repository")
			}
			installedDirs := make([]string, 0, len(installedPackages))
			for _, ps := range installedPackages {
				for _, p := range ps {
					dir := filepath.Join(version.BaseDir(), "pkgs", p.ID)
					installedDirs = append(installedDirs, dir)
				}
			}
			slog.Debug("installedDirs", "dirs", installedDirs)
			installedRepo := repository.NewLocalFileRepository(installedDirs...)
			multirepo.AddRepository(installedRepo)

			packages, err := multirepo.FetchPackages(cmd.Context())
			if err != nil {
				fmt.Fprintf( //nolint:errcheck
					cmd.OutOrStderr(),
					"ERROR: Unable to fetch packages from repositories:\n%v\n",
					err)
				return errors.Wrap(err, "failed to fetch packages from repositories")
			}
			suffix := ""
			if len(packages) == 0 || len(packages) > 1 {
				suffix = "s"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d package%s\n", len(packages), suffix) //nolint:errcheck
			slog.Debug("packages", "packages", packages)

			res := resolver.NewResolverForRepositoryPackages(packages)

			installed, err := state.GetInstalledPackages()
			if err != nil {
				return errors.Wrap(err, "failed to get installed packages")
			}
			resolverInstalled := repoInstalledMapToResolverVPkgMap(installed)

			// parse the human-friendly-ish constraints that remain on the command line
			cliConstraints, err := clicommon.ConstraintsFromArgs(args)
			if err != nil {
				return errors.Wrap(err, "failed to parse package constraints from args")
			}

			// each constraint should remove any packages it matches
			keepInstalledConstraints := make([]*resolver.Constraint, 0, len(installed))
			for _, pkgs := range installed {
				for _, p := range pkgs {
					var ds []*resolver.Constraint
					for _, d := range p.Dependencies {
						ds = append(ds, &resolver.Constraint{
							ID:           resolver.ArtifactID(d.ID),
							Min:          d.Min,
							Max:          d.Max,
							RepositoryID: (*resolver.RepositoryID)(d.RepositoryID),
						})
					}
					vp := &resolver.VersionedPackage{
						ID:            resolver.ArtifactID(p.ID),
						RepositoryID:  "$local-installed",
						Version:       p.Version,
						SupportedArch: p.SupportedArch,
						Dependencies:  ds,
					}
					anyUninstall := false
					for _, cliConstraint := range cliConstraints {
						if cliConstraint.Allows(vp) {
							slog.Debug("will uninstall package", "package", vp, "constraint", cliConstraint)
							anyUninstall = true
						} else {
							slog.Debug("keeping existing package", "package", vp, "constraint", cliConstraint)
						}
					}
					if !anyUninstall {
						resolverID := resolver.RepositoryID(repository.LocalFileRepoID)
						mx := manifest.SemanticVersion{
							Major: vp.Version.Major,
							Minor: vp.Version.Minor,
							Patch: vp.Version.Patch + 1,
						}
						keepInstalledConstraints = append(keepInstalledConstraints, &resolver.Constraint{
							ID:           vp.ID,
							Min:          &vp.Version,
							Max:          &mx,
							RepositoryID: &resolverID,
						})
					}
				}
			}

			slog.Debug("Calling Resolve()",
				"keepInstalledConstraints", keepInstalledConstraints, "cliConstraints", cliConstraints)
			result, err := res.Resolve(keepInstalledConstraints)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "ERROR: Unable to resolve packages:\n%v\n", err) //nolint:errcheck
				return errors.Wrap(err, "failed to resolve packages")
			}

			slog.Debug("resolved packages", "result", result)

			add, rm := resolver.DiffInstallations(resolverInstalled, result)
			if len(rm) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\033[1mPackages to be removed:\033[0m\n") //nolint:errcheck
				for _, art := range rm {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", art) //nolint:errcheck
				}
			}
			if len(add) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\033[1mPackages to be installed:\033[0m\n") //nolint:errcheck
				for _, art := range add {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", art) //nolint:errcheck
				}
			}

			// Sigh, we have to go back to repository.RepoPackage from resolver.Artifact for downloading :(
			addRPs := make([]*repository.RepoPackage, 0, len(add))
			for _, art := range add {
				rp := &repository.RepoPackage{
					ID:            string(art.ID),
					RepositoryID:  string(art.RepositoryID),
					Version:       art.Version,
					Dependencies:  nil,
					SupportedArch: nil,
				}
				addRPs = append(addRPs, rp)
			}
			rmRPs := make([]*repository.RepoPackage, 0, len(rm))
			for _, art := range rm {
				var ds []repository.PackageDependency
				for _, d := range art.Dependencies {
					ds = append(ds, repository.PackageDependency{
						ID:           string(d.ID),
						Min:          d.Min,
						Max:          d.Max,
						RepositoryID: nil,
					})
				}
				rp := &repository.RepoPackage{
					ID:            string(art.ID),
					RepositoryID:  string(art.RepositoryID),
					SupportedArch: art.SupportedArch,
					Version:       art.Version,
					Dependencies:  ds,
				}
				rmRPs = append(rmRPs, rp)
			}

			fmt.Fprint(cmd.OutOrStdout(), "\n\033[1mPerforming package changes...\033[0m\n") //nolint:errcheck
			err = performPackageChanges(ctx, multirepo, addRPs, rmRPs, dryRun)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\033[1mPackages were not installed successfully!\033[0m\n\n") //nolint:errcheck
				return errors.Wrap(err, "failed to install packages")
			}
			return nil
		},
	}
	cmd.Flags().BoolP("dry-run", "n", false, "Perform a trial run with no changes made")
	return cmd
}

func NewInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [flags] example.kpkg",
		Short: "Extract and install a .kpkg file",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return errors.Wrap(err, "failed to get dry-run flag")
			}
			multirepo, err := clicommon.GetRepoFromArgs(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to initialize repository")
			}
			installed, err := state.GetInstalledPackages()
			if err != nil {
				return errors.Wrap(err, "failed to get installed packages")
			}

			installedPackages, err := state.GetInstalledPackages()
			if err != nil {
				return errors.Wrap(err, "failed to get installed packages")
			}
			installedDirs := make([]string, 0, len(installedPackages))
			for _, ps := range installedPackages {
				for _, p := range ps {
					dir := filepath.Join(version.BaseDir(), "pkgs", p.ID)
					installedDirs = append(installedDirs, dir)
				}
			}
			slog.Debug("installedDirs", "dirs", installedDirs)

			fileArgs, rest, err := findFileArgs(args)
			if err != nil {
				return errors.Wrap(err, "failed to parse file arguments")
			}

			// Create a local file repository for the .kpkg files specified on the command line
			// and any extracted (installed, usually) packages
			dirs := make([]string, 0, len(installedDirs)+len(fileArgs))
			dirs = append(dirs, installedDirs...)
			dirs = append(dirs, fileArgs...)
			multirepo.AddRepository(repository.NewLocalFileRepository(dirs...))

			// read metadata from .kpkg files to generate constraints and artifacts
			// used for resolution
			fileConstraints, err := processKPKGArgs(ctx, fileArgs)
			if err != nil {
				return err
			}

			packages, err := multirepo.FetchPackages(cmd.Context())
			if err != nil {
				fmt.Fprintf( //nolint:errcheck
					cmd.OutOrStderr(),
					"ERROR: Unable to fetch packages from repositories:\n%v\n",
					err)
				return errors.Wrap(err, "failed to fetch packages from repositories")
			}
			suffix := ""
			if len(packages) > 1 {
				suffix = "s"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d package%s\n", len(packages), suffix) //nolint:errcheck

			res := resolver.NewResolverForRepositoryPackages(packages)

			// parse the human-friendly-ish constraints that remain on the command line
			constraints, err := clicommon.ConstraintsFromArgs(rest)
			if err != nil {
				return errors.Wrap(err, "failed to parse package constraints from args")
			}

			constraints = append(fileConstraints, constraints...)

			result, err := res.Resolve(constraints)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "ERROR: Unable to resolve packages:\n%v\n", err) //nolint:errcheck
				return errors.Wrap(err, "failed to resolve packages")
			}

			slog.Debug("resolved packages", "result", result)

			resolverInstalled := repoInstalledMapToResolverVPkgMap(installed)
			add, rm := resolver.DiffInstallations(resolverInstalled, result)
			if len(rm) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\033[1mPackages to be removed:\033[0m\n") //nolint:errcheck
				for _, art := range rm {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", art) //nolint:errcheck
				}
			}
			if len(add) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\033[1mPackages to be installed:\033[0m\n") //nolint:errcheck
				for _, art := range add {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", art) //nolint:errcheck
				}
			}

			// Sigh, we have to go back to repository.RepoPackage from resolver.Artifact for downloading :(
			addRPs := make([]*repository.RepoPackage, 0, len(add))
			for _, art := range add {
				rp := &repository.RepoPackage{
					ID:            string(art.ID),
					RepositoryID:  string(art.RepositoryID),
					Version:       art.Version,
					Dependencies:  nil,
					SupportedArch: nil,
				}
				addRPs = append(addRPs, rp)
			}
			rmRPs := make([]*repository.RepoPackage, 0, len(rm))
			for _, art := range rm {
				var ds []repository.PackageDependency
				for _, d := range art.Dependencies {
					ds = append(ds, repository.PackageDependency{
						ID:           string(d.ID),
						Min:          d.Min,
						Max:          d.Max,
						RepositoryID: nil,
					})
				}
				rp := &repository.RepoPackage{
					ID:            string(art.ID),
					RepositoryID:  string(art.RepositoryID),
					SupportedArch: art.SupportedArch,
					Version:       art.Version,
					Dependencies:  ds,
				}
				rmRPs = append(rmRPs, rp)
			}

			fmt.Fprint(cmd.OutOrStdout(), "\n\033[1mPerforming package changes...\033[0m\n") //nolint:errcheck
			err = performPackageChanges(ctx, multirepo, addRPs, rmRPs, dryRun)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\033[1mPackages were not installed successfully!\033[0m\n\n") //nolint:errcheck
				return errors.Wrap(err, "failed to install packages")
			}
			return nil
		},
	}
	cmd.Flags().BoolP("dry-run", "n", false, "Perform a trial run with no changes made")
	return cmd
}

func performPackageChanges(
	ctx context.Context, repo repository.Repository, add, rm []*repository.RepoPackage, dryRun bool,
) error {
	slog.Debug("performPackageChanges()", "repo", repo.ID(), "add", len(add), "remove", len(rm), "dryRun", dryRun)
	if len(rm) > 0 {
		for _, rp := range rm {
			err := removePackage(ctx, rp, dryRun)
			if err != nil {
				return err
			}
		}
	}
	if len(add) > 0 {
		for _, rp := range add {
			err := addPackage(ctx, repo, rp, dryRun)
			if err != nil {
				return err
			}
			fmt.Printf("\033[1m%s:\033[0m installed successfully\n", rp.ID)
		}
	}
	if dryRun {
		fmt.Println("\n\033[1mDry run finished! No changes were made.\033[0m")
		return nil
	}
	return nil
}

func removePackage(ctx context.Context, rp *repository.RepoPackage, dryRun bool) error {
	// TODO: rewrite this to duplicate less with addPackage
	var err error
	pkgDirName := rp.ID
	baseDir := version.BaseDir()
	pkgsDir := filepath.Join(baseDir, "pkgs")
	destDir := filepath.Join(pkgsDir, pkgDirName)
	uninstallerPath := filepath.Join(destDir, "uninstall.sh")

	fmt.Printf("Running uninstall script for %s (version %s)\n", rp.ID, rp.Version.String())

	cmd := exec.CommandContext(ctx, "/bin/sh", "-l", uninstallerPath)
	cmd.Dir = destDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dryRun {
		fmt.Printf(" - [dry-run] /bin/sh -l %q\n", uninstallerPath)
		err = nil
	} else {
		err = cmd.Run()
	}
	if err != nil {
		return fmt.Errorf("failed to remove package dir %q: %w", destDir, err)
	}
	if dryRun {
		fmt.Printf(" - [dry-run] Removed package directory %q\n", destDir)
	} else {
		err = os.RemoveAll(destDir)
		if err != nil {
			return fmt.Errorf("failed to remove package dir %q: %w", destDir, err)
		}
	}
	return nil
}

// this is all begging to be refactored elsewhere

func downloadAndUnpack(
	ctx context.Context, repo repository.Repository, rp *repository.RepoPackage, destDir string, dryRun bool,
) error {
	if dryRun {
		fmt.Printf(" - [dry-run] Downloading and unpacking package %s to %s\n", rp, destDir)
		return nil
	}

	err := os.MkdirAll(destDir, 0o755) //nolint:gosec
	if err != nil {
		return errors.AddStack(err)
	}

	tmpFile, err := os.CreateTemp("", "*.kpkg")
	if err != nil {
		return errors.Wrapf(err, "os.CreateTemp()")
	}
	defer func() {
		_ = os.RemoveAll(tmpFile.Name())
	}()

	kpkgPath := tmpFile.Name()
	slog.Debug("downloadPackage()", "kpkgPath", kpkgPath)
	err = repo.DownloadPackage(ctx, rp, kpkgPath, dryRun)
	if err != nil {
		return errors.Wrapf(err, "repo.DownloadPackage(%q)", kpkgPath)
	}

	if dryRun {
		fmt.Printf(" - [dry-run] Unpacked package %s to %s\n", rp, destDir)
		return nil
	}
	kpkgFile, err := kpkg.Open(ctx, kpkgPath)
	if err != nil {
		return errors.Wrapf(err, "kpkg.Open(%q)", kpkgPath)
	}
	defer func() { _ = kpkgFile.Close() }()

	tmpDir, err := os.MkdirTemp("", "kpm-extract-"+kpkgFile.Manifest.ID)
	if err != nil {
		return errors.Wrapf(err, "os.MkdirTemp()")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	slog.Debug("extracting KPKG", "kpkg", kpkgPath, "destDir", tmpDir, "package", kpkgFile.Manifest)

	err = kpkgFile.ExtractAll(ctx, tmpDir, false, os.Stdout)
	if err != nil {
		return errors.Wrapf(err, "kpkg.ExtractAll(%q, %q)", rp, tmpDir)
	}
	err = copyDirSafe(tmpDir, destDir)
	if err != nil {
		return errors.Wrapf(err, "copyDirSafe(%q, %q)", tmpDir, destDir)
	}

	return nil
}

func copyDirSafe(srcDir, destDir string) error {
	srcFS := os.DirFS(srcDir)
	err := fs.WalkDir(srcFS, ".", func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.AddStack(err)
		}
		destPath := filepath.Join(destDir, srcPath)
		slog.Debug("copyDirSafe()", "srcPath", srcPath, "destPath", destPath)

		//nolint:exhaustive
		switch d.Type() & fs.ModeType {
		case fs.ModeSymlink:
			slog.Warn("link copying is not supported on /mnt/us, skipping", "path", srcPath)
			return nil
		case fs.ModeDir:
			err := os.MkdirAll(destPath, 0o755) //nolint:gosec
			if err != nil {
				return errors.Wrapf(err, "os.MkdirAll(%q)", destPath)
			}
			return nil
		case 0: // regular file
			srcFile, err := os.Open(filepath.Join(srcDir, srcPath))
			if err != nil {
				return errors.Wrapf(err, "os.Open(%q)", srcPath)
			}
			defer srcFile.Close()

			destFile, err := os.Create(destPath)
			if err != nil {
				return errors.Wrapf(err, "os.Create(%q)", destPath)
			}
			defer destFile.Close()

			_, err = io.Copy(destFile, srcFile)
			if err != nil {
				return errors.Wrapf(err, "io.Copy(%q, %q)", srcPath, destPath)
			}
		default:
			slog.Warn("unsupported file type, skipping", "path", srcPath, "type", d.Type())
		}
		return nil
	})
	return errors.AddStack(err)
}

func addPackage(ctx context.Context, repo repository.Repository, rp *repository.RepoPackage, dryRun bool) error {
	baseDir := version.BaseDir()
	// TODO: is this desirable? It means you can't assume you're in /mnt/us/kpm/pkgs/$name/, which
	// could be useful if absolute paths are needed somewhere.
	// pkgDirName := fmt.Sprintf("%s-%d.%d.%d", rp.ID, rp.Version.Major, rp.Version.Minor, rp.Version.Patch)
	pkgDirName := rp.ID
	pkgsDir := filepath.Join(baseDir, "pkgs")
	destDir := filepath.Join(pkgsDir, pkgDirName)

	slog.Debug("downloadAndUnpack()", "rp", rp, "destDir", destDir, "dryRun", dryRun)
	err := downloadAndUnpack(ctx, repo, rp, destDir, dryRun)
	if err != nil {
		return errors.Wrapf(err, "failed to stage package %s", rp)
	}

	installerPath := destDir + "/install.sh"
	_, err = os.Stat(installerPath)
	if os.IsNotExist(err) {
		slog.Debug("no install script for package %q", "path", rp.ID)
		return nil
	}
	// TODO: add test for this behavior
	err = os.Chmod(installerPath, 0o755) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to make installer %q executable: %w", installerPath, err)
	}

	fmt.Printf("Running install script for %s (version %s)\n", rp.ID, rp.Version.String())

	cmd := exec.CommandContext(ctx, "/bin/sh", "-l", installerPath)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "KPM_INSTALL_DIR="+destDir)
	cmd.Env = append(cmd.Env, "KPM_BASE_DIR="+baseDir)
	cmd.Env = append(cmd.Env, "KPM_USERSTORE_DIR="+version.UserstoreDir())
	cmd.Dir = destDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dryRun {
		fmt.Printf(" - [dry-run] /bin/sh -l %q\n", installerPath)
	} else {
		err = cmd.Run()
	}
	if err != nil {
		return fmt.Errorf("failed to install package %q: %w", installerPath, err)
	}
	return nil
}

func processKPKGArgs(ctx context.Context, fileArgs []string) ([]*resolver.Constraint, error) {
	var manifests []*manifest.Manifest
	for _, f := range fileArgs {
		kpkg, err := kpkg.Open(ctx, f)
		if err != nil {
			return nil, errors.Wrapf(err, "kpkg.OpenKPKGFile(%q)", f)
		}
		defer kpkg.Close() //nolint:errcheck
		pkgManifest := kpkg.Manifest
		if pkgManifest == nil {
			return nil, fmt.Errorf("kpkg %q has no manifest", f)
		}
		manifests = append(manifests, pkgManifest)
	}

	constraints, err := constraintsFromKPKGFiles(manifests)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate constraints from .kpkg files")
	}

	return constraints, nil
}

// findFileArgs separates .kpkg file arguments from version constraint (foo=1.2.3) arguments.
func findFileArgs(args []string) ([]string, []string, error) {
	var fileArgs []string
	var rest []string
	for _, arg := range args {
		fi, _ := os.Stat(arg)
		exists := fi != nil && fi.Mode().IsRegular()

		if strings.HasSuffix(arg, ".kpkg") || exists {
			// if you try to pass a .kpkg file, it's an error to not exist (in other words,
			// we are explicitly skipping parsing it as a package name)
			if !exists {
				return nil, nil, fmt.Errorf("file %q does not exist", arg)
			}
			fileArgs = append(fileArgs, arg)
		} else {
			rest = append(rest, arg)
		}
	}
	return fileArgs, rest, nil
}

// constraintsFromKPKGFiles generates constraints from the given .kpkg files,
// including dependencies specified in their manifests.
func constraintsFromKPKGFiles(manifests []*manifest.Manifest) ([]*resolver.Constraint, error) {
	var constraints []*resolver.Constraint
	for _, pkgManifest := range manifests {
		cs, err := func() ([]*resolver.Constraint, error) { //nolint:unparam
			maxC := manifest.SemanticVersion{
				Major: pkgManifest.Version.Major,
				Minor: pkgManifest.Version.Minor,
				Patch: pkgManifest.Version.Patch + 1,
			}
			constraint := &resolver.Constraint{
				ID:           resolver.ArtifactID(pkgManifest.ID),
				RepositoryID: nil,
				Min:          &pkgManifest.Version,
				Max:          &maxC,
			}

			cs := []*resolver.Constraint{constraint}
			for depID, dep := range pkgManifest.Dependencies {
				cs = append(cs, &resolver.Constraint{
					ID:           resolver.ArtifactID(depID),
					RepositoryID: nil,
					Min:          dep.Min,
					Max:          dep.Max,
				})
			}
			return cs, nil
		}()
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, cs...)
	}
	return constraints, nil
}
