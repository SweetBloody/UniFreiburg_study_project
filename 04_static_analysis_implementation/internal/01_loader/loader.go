package loader

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// LoadPackages loads the target Go package with syntax and type information
func LoadPackages(pattern string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages, error: %w", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package loading errors")
	}

	return pkgs, nil
}
