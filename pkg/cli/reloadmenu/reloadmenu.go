package reloadmenu

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/state"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reload-menu",
		Short: "Regenerate the KUAL menu.json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			installedPkgs, err := state.GetInstalledPackages()
			if err != nil {
				return errors.AddStack(err)
			}

			menu := generateMenuJSON(installedPkgs)
			if err != nil {
				return errors.AddStack(err)
			}
			menuJSON, err := json.MarshalIndent(menu, "", "  ")
			if err != nil {
				return errors.AddStack(err)
			}

			write, err := cmd.Flags().GetBool("write")
			if err != nil {
				return errors.AddStack(err)
			}
			if write {
				menuPath := filepath.Join(version.UserstoreDir(), "extensions", "kpmgo", "menu.json")
				f, err := os.Create(menuPath)
				if err != nil {
					return errors.Wrapf(err, "os.Create(%q)", menuPath)
				}
				defer f.Close()
				_, err = f.Write(menuJSON)
				if err != nil {
					return errors.Wrapf(err, "writing regenerated menu.json to %q", menuPath)
				}
				slog.Info("regenerated menu.json written", "path", menuPath)
			} else {
				_, err = cmd.OutOrStdout().Write(menuJSON)
				if err != nil {
					return errors.AddStack(err)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolP("write", "w", false, "Write the regenerated menu.json to disk")
	return cmd
}

type KUALMenu struct {
	Items []*KUALMenuItem `json:"items"`
}

type KUALMenuItem struct {
	Name     string          `json:"name"`
	Action   string          `json:"action,omitempty"`
	Params   string          `json:"params,omitempty"`
	ExitMenu bool            `json:"exitmenu"`
	Checked  bool            `json:"checked,omitempty"`
	Internal string          `json:"internal,omitempty"`
	Refresh  bool            `json:"refresh,omitempty"`
	Items    []*KUALMenuItem `json:"items,omitempty"`
}

//nolint:exhaustruct
func generateMenuJSON(installedPkgs map[string][]*repository.RepoPackage) *KUALMenu {
	rootMenu := &KUALMenu{}
	menu := KUALMenuItem{
		Name:  "kpmgo",
		Items: nil,
	}
	rootMenu.Items = append(rootMenu.Items, &menu)

	menu.Items = append(menu.Items, &KUALMenuItem{
		Name:     "Reload menu",
		Action:   "./extension.sh",
		Params:   "reload-menu",
		ExitMenu: false,
		Refresh:  true,
	})
	slog.Info("generating menu for installed packages", "count", len(installedPkgs), "menu", menu.Items)

	modifyItems := []*KUALMenuItem{}
	for pkgID := range installedPkgs {
		modifyItems = append(modifyItems, &KUALMenuItem{
			Name:     "Uninstall " + pkgID,
			Action:   "./extension.sh",
			Params:   "uninstall " + pkgID,
			Checked:  true,
			ExitMenu: false,
			Refresh:  true,
			Internal: "status Uninstalling " + pkgID + "...",
		})
	}
	menu.Items = append(menu.Items, &KUALMenuItem{
		Name:  "Modify installed packages",
		Items: modifyItems,
	})

	menu.Items = append(menu.Items, getLaunchItems(installedPkgs)...)

	return rootMenu
}

func getLaunchItems(installedPkgs map[string][]*repository.RepoPackage) []*KUALMenuItem {
	launchItems := []*KUALMenuItem{}
	for pkgID := range installedPkgs {
		launchPath := filepath.Join(version.BaseDir(), "pkgs", pkgID, "launch.sh")
		_, err := os.Stat(launchPath)
		if err != nil {
			slog.Debug("no launch.sh for package, skipping launch menu item", "package", pkgID, "path", launchPath)
			continue
		}
		launchItems = append(launchItems, &KUALMenuItem{ //nolint:exhaustruct
			Name:     "Launch " + pkgID,
			Action:   "./extension.sh",
			Params:   "launch " + pkgID,
			ExitMenu: true,
			Internal: "status Launching " + pkgID + "...",
		})
	}
	return launchItems
}
