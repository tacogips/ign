package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/app"
	"github.com/tacogips/ign/internal/template/model"
)

func TestVarsCommandListsTemplateVariables(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := app.Init(context.Background(), app.InitOptions{URL: templatePath}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	beforeIgnConfig := readFileString(t, filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile))
	beforeIgnVar := readFileString(t, filepath.Join(model.IgnConfigDir, model.IgnVarFile))

	result, err := app.InspectVars(context.Background(), app.VarsOptions{})
	if err != nil {
		t.Fatalf("InspectVars failed: %v", err)
	}
	if !result.DeclarationsAvailable {
		t.Fatalf("declarations unavailable: %v", result.DeclarationError)
	}

	rows := integrationRowsByName(result.Rows)
	for _, name := range []string{"project_name", "port", "enable_feature"} {
		if _, ok := rows[name]; !ok {
			t.Fatalf("missing vars row %q in %#v", name, result.Rows)
		}
	}

	if got := readFileString(t, filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)); got != beforeIgnConfig {
		t.Fatal("InspectVars mutated .ign/ign.json")
	}
	if got := readFileString(t, filepath.Join(model.IgnConfigDir, model.IgnVarFile)); got != beforeIgnVar {
		t.Fatal("InspectVars mutated .ign/ign-var.json")
	}
}

func TestVarsCommandUnsetRequiredMissing(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := app.Init(context.Background(), app.InitOptions{URL: templatePath}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}
	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}
	delete(ignVar.Variables, "project_name")
	updated, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}
	if err := os.WriteFile(ignVarPath, updated, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	result, err := app.InspectVars(context.Background(), app.VarsOptions{})
	if err != nil {
		t.Fatalf("InspectVars failed: %v", err)
	}
	unset := app.FilterUnsetVars(result)
	if unset.RequiredUnsetCount != 1 {
		t.Fatalf("RequiredUnsetCount = %d, want 1", unset.RequiredUnsetCount)
	}
	if len(unset.Rows) != 1 || unset.Rows[0].Name != "project_name" {
		t.Fatalf("unset rows = %#v, want project_name only", unset.Rows)
	}
}

func TestVarsCLICommandTableJSONAndUnsetExit(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)
	ignBinary := buildIgnBinary(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := app.Init(context.Background(), app.InitOptions{URL: templatePath}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	table := runIgnCLI(t, ignBinary, tempDir, "vars")
	if table.exitCode != 0 {
		t.Fatalf("ign vars exit = %d, stderr = %s", table.exitCode, table.stderr)
	}
	for _, want := range []string{"NAME", "TYPE", "REQUIRED", "DEFAULT", "CURRENT", "DESCRIPTION", "project_name"} {
		if !strings.Contains(table.stdout, want) {
			t.Fatalf("ign vars stdout %q does not contain %q", table.stdout, want)
		}
	}

	jsonRun := runIgnCLI(t, ignBinary, tempDir, "vars", "--json")
	if jsonRun.exitCode != 0 {
		t.Fatalf("ign vars --json exit = %d, stderr = %s", jsonRun.exitCode, jsonRun.stderr)
	}
	var decoded app.VarsResult
	if err := json.Unmarshal([]byte(jsonRun.stdout), &decoded); err != nil {
		t.Fatalf("ign vars --json stdout is not valid JSON: %v\n%s", err, jsonRun.stdout)
	}
	if len(decoded.Rows) == 0 {
		t.Fatal("ign vars --json returned no rows")
	}

	unsetSatisfied := runIgnCLI(t, ignBinary, tempDir, "vars", "--unset")
	if unsetSatisfied.exitCode != 0 {
		t.Fatalf("ign vars --unset exit = %d, stderr = %s", unsetSatisfied.exitCode, unsetSatisfied.stderr)
	}
	if strings.Contains(unsetSatisfied.stdout, "project_name") {
		t.Fatalf("ign vars --unset stdout = %q, want no satisfied variables", unsetSatisfied.stdout)
	}

	removeIgnVar(t, "project_name")

	unsetMissing := runIgnCLI(t, ignBinary, tempDir, "vars", "--unset")
	if unsetMissing.exitCode != 1 {
		t.Fatalf("ign vars --unset missing exit = %d, stderr = %s", unsetMissing.exitCode, unsetMissing.stderr)
	}
	if !strings.Contains(unsetMissing.stdout, "project_name") {
		t.Fatalf("ign vars --unset missing stdout = %q, want project_name", unsetMissing.stdout)
	}

	jsonUnsetMissing := runIgnCLI(t, ignBinary, tempDir, "vars", "--json", "--unset")
	if jsonUnsetMissing.exitCode != 1 {
		t.Fatalf("ign vars --json --unset missing exit = %d, stderr = %s", jsonUnsetMissing.exitCode, jsonUnsetMissing.stderr)
	}
	var decodedUnset app.VarsResult
	if err := json.Unmarshal([]byte(jsonUnsetMissing.stdout), &decodedUnset); err != nil {
		t.Fatalf("ign vars --json --unset stdout is not valid JSON: %v\n%s", err, jsonUnsetMissing.stdout)
	}
	if decodedUnset.RequiredUnsetCount != 1 {
		t.Fatalf("RequiredUnsetCount = %d, want 1", decodedUnset.RequiredUnsetCount)
	}
}

func integrationRowsByName(rows []app.VarsRow) map[string]app.VarsRow {
	result := make(map[string]app.VarsRow, len(rows))
	for _, row := range rows {
		result[row.Name] = row
	}
	return result
}

func readFileString(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}

func removeIgnVar(t *testing.T, name string) {
	t.Helper()

	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}
	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}
	delete(ignVar.Variables, name)
	updated, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}
	if err := os.WriteFile(ignVarPath, updated, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}
}

func buildIgnBinary(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate integration test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	binaryPath := filepath.Join(t.TempDir(), "ign-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/ign")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build ign binary: %v\n%s", err, string(output))
	}
	return binaryPath
}

type ignCLIResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runIgnCLI(t *testing.T, binaryPath, workDir string, args ...string) ignCLIResult {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN=", "GH_TOKEN=", "PATH=/nonexistent")
	stdout, err := cmd.Output()
	stderr := ""
	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr = string(exitErr.Stderr)
		return ignCLIResult{stdout: string(stdout), stderr: stderr, exitCode: exitErr.ExitCode()}
	}
	if err != nil {
		t.Fatalf("failed to run ign %v: %v", args, err)
	}
	return ignCLIResult{stdout: string(stdout), stderr: stderr, exitCode: 0}
}
