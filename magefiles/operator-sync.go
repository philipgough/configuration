package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strconv"

	"github.com/magefile/mage/mg"
	"github.com/rhobs/configuration/clusters"
	"github.com/rhobs/configuration/internal/submodule"
)

type (
	Sync mg.Namespace
)

// Operator syncs the given operator manifests from the specified ref.
// Ref can be a specific commit or "latest" to use the latest commit from upstream.
func (s Sync) Operator(operator string, ref string) error {
	if operator != "thanos" {
		return fmt.Errorf("unsupported operator: %s", operator)
	}

	if err := s.syncThanosOperator(ref); err != nil {
		return fmt.Errorf("failed to sync Thanos Operator: %w", err)
	}
	return nil
}

// ThanosOperatorRef syncs Thanos Operator manifests from the given ref.
func (s Sync) syncThanosOperator(ref string) error {
	const (
		apiURL = "https://api.github.com/repos/rhobs/rhobs-konflux-thanos-operator"
	)

	if ref == "latest" {
		latestMainSHA, err := submodule.GithubLatestCommit(apiURL)
		if err != nil {
			return fmt.Errorf("failed to get latest commit from main branch: %w", err)
		}
		ref = latestMainSHA
		fmt.Fprintf(os.Stdout, "No ref provided, using latest commit from main: %s\n", ref)
	}

	fmt.Fprintf(os.Stdout, "Syncing Thanos Operator at commit %s\n", ref)
	err := updateConst("ThanosOperatorVersion", "clusters/template.go", ref)
	if err != nil {
		return fmt.Errorf("failed to update Thanos Operator ref: %w", err)
	}

	const (
		repoURL       = "https://github.com/rhobs/rhobs-konflux-thanos-operator"
		submodulePath = "thanos-operator"
	)

	info := submodule.Info{
		Commit:        clusters.ThanosOperatorVersion,
		SubmodulePath: submodulePath,
		URL:           repoURL,
	}
	crdRef, err := info.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse submodule info: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Parsed submodule ref: %+v\n", crdRef)
	err = updateConst("thanosOperatorCRDRef", "magefiles/thanos-operator.go", crdRef)
	if err != nil {
		return fmt.Errorf("failed to update Thanos Operator CRD ref: %w", err)
	}

	// Update go.mod with the CRD reference
	err = updateGoMod(crdRef)
	if err != nil {
		return fmt.Errorf("failed to update go.mod: %w", err)
	}

	return nil
}

func updateGoMod(crdRef string) error {
	const thanosOperatorModule = "github.com/thanos-community/thanos-operator"

	// Use go mod edit to update the thanos-operator dependency to the specific commit
	cmd := exec.Command("go", "mod", "edit", "-require", fmt.Sprintf("%s@%s", thanosOperatorModule, crdRef))
	fmt.Fprintf(os.Stdout, "Running: %s\n", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update go.mod: %w\nOutput: %s", err, string(output))
	}

	// Run go mod tidy to clean up
	cmd = exec.Command("go", "mod", "tidy")
	fmt.Fprintf(os.Stdout, "Running: %s\n", cmd.String())

	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w\nOutput: %s", err, string(output))
	}

	fmt.Fprintf(os.Stdout, "Updated go.mod: %s@%s\n", thanosOperatorModule, crdRef)
	return nil
}

func updateConst(constName, inFile, newRef string) error {
	filename := inFile
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Find and update the thanosOperatorRef constant
	ast.Inspect(node, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range valueSpec.Names {
						if name.Name == constName && i < len(valueSpec.Values) {
							if basicLit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
								basicLit.Value = strconv.Quote(newRef)
								return false // Stop searching
							}
						}
					}
				}
			}
		}
		return true
	})

	// Write the updated AST back to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := format.Node(file, fset, node); err != nil {
		return fmt.Errorf("failed to write formatted code: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Updated %s to: %s\n", constName, newRef)
	return nil
}
