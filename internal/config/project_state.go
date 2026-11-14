package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

var renameStagedProjectStateFile = os.Rename

// ProjectState is the normalized project-side .ign metadata view.
type ProjectState struct {
	Version    int
	OutputRoot string
	Templates  []ProjectTemplateState
}

// ProjectTemplateState is one checked-out template plus its variables and files.
type ProjectTemplateState struct {
	ID        string
	Order     int
	Template  model.TemplateSource
	Hash      string
	Metadata  *model.FileMetadata
	Variables map[string]interface{}
	Files     []string
}

// LoadProjectState loads .ign metadata and normalizes v1 or v2 into ProjectState.
func LoadProjectState(configDir string) (*ProjectState, error) {
	ignConfigPath := filepath.Join(configDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(configDir, model.IgnVarFile)
	manifestPath := filepath.Join(configDir, model.IgnManifestFile)

	ignConfig, err := LoadIgnConfig(ignConfigPath)
	if err != nil {
		return nil, err
	}
	ignVar, err := LoadIgnVarJson(ignVarPath)
	if err != nil {
		return nil, err
	}
	manifest, manifestErr := LoadIgnManifest(manifestPath)
	if manifestErr != nil {
		if cfgErr, ok := manifestErr.(*ConfigError); !ok || cfgErr.Type != ConfigNotFound {
			return nil, manifestErr
		}
		manifest = &model.IgnManifest{}
	}

	configVersion := metadataVersion(ignConfig.Version)
	varVersion := metadataVersion(ignVar.Version)
	manifestVersion := metadataVersion(manifest.Version)
	if configVersion != varVersion || configVersion != manifestVersion {
		return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "version", "metadata file versions must match")
	}

	switch configVersion {
	case model.IgnProjectMetadataVersionV1:
		return normalizeProjectStateV1(ignConfig, ignVar, manifest), nil
	case model.IgnProjectMetadataVersionV2:
		return normalizeProjectStateV2(configDir, ignConfig, ignVar, manifest)
	default:
		return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "version", fmt.Sprintf("unsupported metadata version %d", configVersion))
	}
}

// SaveProjectState saves ProjectState as version 2 metadata files.
func SaveProjectState(configDir string, state *ProjectState) error {
	if state == nil {
		return NewConfigErrorWithField(ConfigValidationFailed, configDir, "", "project state cannot be nil")
	}
	normalized := normalizeOutputRoot(state.OutputRoot)
	if normalized == "" {
		normalized = "."
	}

	templates := append([]ProjectTemplateState(nil), state.Templates...)
	sort.SliceStable(templates, func(i, j int) bool {
		return templates[i].Order < templates[j].Order
	})

	ignConfig := &model.IgnConfig{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: normalized,
		Templates:  make([]model.TemplateCheckoutRecord, 0, len(templates)),
	}
	ignVar := &model.IgnVarJson{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: normalized,
		Templates:  make([]model.IgnVarTemplateRecord, 0, len(templates)),
	}
	manifest := &model.IgnManifest{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: normalized,
		Templates:  make([]model.IgnManifestTemplateRecord, 0, len(templates)),
	}

	for _, tmpl := range templates {
		ignConfig.Templates = append(ignConfig.Templates, model.TemplateCheckoutRecord{
			ID:       tmpl.ID,
			Order:    tmpl.Order,
			Template: tmpl.Template,
			Hash:     tmpl.Hash,
			Metadata: tmpl.Metadata,
		})
		vars := tmpl.Variables
		if vars == nil {
			vars = map[string]interface{}{}
		}
		ignVar.Templates = append(ignVar.Templates, model.IgnVarTemplateRecord{
			ID:        tmpl.ID,
			Variables: vars,
		})
		manifest.Templates = append(manifest.Templates, model.IgnManifestTemplateRecord{
			ID:    tmpl.ID,
			Order: tmpl.Order,
			Files: cleanManifestFiles(tmpl.Files),
		})
	}
	if len(templates) > 0 {
		first := templates[0]
		ignConfig.Template = first.Template
		ignConfig.Hash = first.Hash
		ignConfig.Metadata = first.Metadata
		ignVar.Variables = first.Variables
		for _, tmpl := range templates {
			manifest.Files = append(manifest.Files, tmpl.Files...)
		}
		manifest.Files = cleanManifestFiles(manifest.Files)
	}

	if err := validateProjectStateForSave(configDir, normalized, templates); err != nil {
		return err
	}

	writes := []struct {
		path string
		data interface{}
	}{
		{filepath.Join(configDir, model.IgnProjectConfigFile), ignConfig},
		{filepath.Join(configDir, model.IgnVarFile), ignVar},
		{filepath.Join(configDir, model.IgnManifestFile), manifest},
	}
	return writeProjectStateFiles(writes)
}

type projectStateWrite struct {
	path        string
	data        interface{}
	tmpPath     string
	backupPath  string
	hadExisting bool
}

func writeProjectStateFiles(writes []struct {
	path string
	data interface{}
}) error {
	staged := make([]projectStateWrite, 0, len(writes))
	for _, write := range writes {
		next, err := stageProjectStateFile(write.path, write.data)
		if err != nil {
			cleanupProjectStateTemps(staged)
			return err
		}
		staged = append(staged, next)
	}
	if err := replaceProjectStateFiles(staged); err != nil {
		cleanupProjectStateTemps(staged)
		return err
	}
	return nil
}

func stageProjectStateFile(path string, value interface{}) (projectStateWrite, error) {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return projectStateWrite{}, NewConfigErrorWithCause(ConfigInvalid, path, "path contains '..' which is not allowed for security reasons", nil)
	}
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return projectStateWrite{}, NewConfigErrorWithCause(ConfigInvalid, cleanPath, fmt.Sprintf("failed to create directory %s", dir), err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return projectStateWrite{}, NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to marshal JSON", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(cleanPath)+".tmp-*")
	if err != nil {
		return projectStateWrite{}, NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to create temporary file", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return projectStateWrite{}, NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to write temporary file", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return projectStateWrite{}, NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to close temporary file", err)
	}
	return projectStateWrite{path: cleanPath, data: value, tmpPath: tmpPath}, nil
}

func replaceProjectStateFiles(writes []projectStateWrite) error {
	for i := range writes {
		info, err := os.Stat(writes[i].path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			restoreProjectStateBackups(writes)
			return NewConfigErrorWithCause(ConfigInvalid, writes[i].path, "failed to stat existing metadata file", err)
		}
		if info.IsDir() {
			restoreProjectStateBackups(writes)
			return NewConfigErrorWithCause(ConfigInvalid, writes[i].path, "metadata path is a directory", nil)
		}
		backup, err := createProjectStateBackupPath(writes[i].path)
		if err != nil {
			restoreProjectStateBackups(writes)
			return err
		}
		if err := os.Rename(writes[i].path, backup); err != nil {
			_ = os.Remove(backup)
			restoreProjectStateBackups(writes)
			return NewConfigErrorWithCause(ConfigInvalid, writes[i].path, "failed to backup existing metadata file", err)
		}
		writes[i].backupPath = backup
		writes[i].hadExisting = true
	}

	installed := []projectStateWrite{}
	for i := range writes {
		if err := renameStagedProjectStateFile(writes[i].tmpPath, writes[i].path); err != nil {
			_ = restoreProjectStateFiles(writes, installed)
			return NewConfigErrorWithCause(ConfigInvalid, writes[i].path, "failed to replace metadata file set", err)
		}
		installed = append(installed, writes[i])
		writes[i].tmpPath = ""
	}

	for _, write := range writes {
		if write.hadExisting {
			_ = os.Remove(write.backupPath)
		}
	}
	return nil
}

func createProjectStateBackupPath(path string) (string, error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".bak-*")
	if err != nil {
		return "", NewConfigErrorWithCause(ConfigInvalid, path, "failed to create metadata backup file", err)
	}
	backup := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(backup)
		return "", NewConfigErrorWithCause(ConfigInvalid, path, "failed to close metadata backup file", err)
	}
	if err := os.Remove(backup); err != nil {
		return "", NewConfigErrorWithCause(ConfigInvalid, path, "failed to prepare metadata backup file", err)
	}
	return backup, nil
}

func restoreProjectStateFiles(writes []projectStateWrite, installed []projectStateWrite) error {
	errs := []error{}
	for _, write := range installed {
		if err := os.Remove(write.path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	for _, write := range writes {
		if !write.hadExisting {
			continue
		}
		if err := os.Rename(write.backupPath, write.path); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func restoreProjectStateBackups(writes []projectStateWrite) {
	_ = restoreProjectStateFiles(writes, nil)
}

func cleanupProjectStateTemps(writes []projectStateWrite) {
	for _, write := range writes {
		if write.tmpPath != "" {
			_ = os.Remove(write.tmpPath)
		}
	}
}

func metadataVersion(version int) int {
	if version == 0 {
		return model.IgnProjectMetadataVersionV1
	}
	return version
}

func normalizeProjectStateV1(ignConfig *model.IgnConfig, ignVar *model.IgnVarJson, manifest *model.IgnManifest) *ProjectState {
	id := templateRecordIDFromMetadata(ignConfig.Metadata, 1)
	vars := ignVar.Variables
	if vars == nil {
		vars = map[string]interface{}{}
	}
	return &ProjectState{
		Version:    model.IgnProjectMetadataVersionV1,
		OutputRoot: ".",
		Templates: []ProjectTemplateState{
			{
				ID:        id,
				Order:     1,
				Template:  ignConfig.Template,
				Hash:      ignConfig.Hash,
				Metadata:  ignConfig.Metadata,
				Variables: vars,
				Files:     cleanManifestFiles(manifest.Files),
			},
		},
	}
}

func normalizeProjectStateV2(configDir string, ignConfig *model.IgnConfig, ignVar *model.IgnVarJson, manifest *model.IgnManifest) (*ProjectState, error) {
	if err := validateProjectStateOutputRoot(configDir, ignConfig.OutputRoot); err != nil {
		return nil, err
	}
	if err := validateProjectStateOutputRoot(configDir, ignVar.OutputRoot); err != nil {
		return nil, err
	}
	if err := validateProjectStateOutputRoot(configDir, manifest.OutputRoot); err != nil {
		return nil, err
	}
	outputRoot := normalizeOutputRoot(ignConfig.OutputRoot)
	if outputRoot != normalizeOutputRoot(ignVar.OutputRoot) || outputRoot != normalizeOutputRoot(manifest.OutputRoot) {
		return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "output_root", "output_root must match across metadata files")
	}
	if len(ignConfig.Templates) == 0 {
		return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates", "at least one template record is required")
	}

	varsByID := make(map[string]map[string]interface{}, len(ignVar.Templates))
	for _, record := range ignVar.Templates {
		if record.ID == "" {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.id", "variable template id is required")
		}
		varsByID[record.ID] = record.Variables
	}
	filesByID := make(map[string][]string, len(manifest.Templates))
	orderByManifestID := make(map[string]int, len(manifest.Templates))
	for _, record := range manifest.Templates {
		if record.ID == "" {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.id", "manifest template id is required")
		}
		filesByID[record.ID] = cleanManifestFiles(record.Files)
		orderByManifestID[record.ID] = record.Order
	}

	seen := make(map[string]struct{}, len(ignConfig.Templates))
	templates := make([]ProjectTemplateState, 0, len(ignConfig.Templates))
	for _, record := range ignConfig.Templates {
		if record.ID == "" {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.id", "template id is required")
		}
		if _, ok := seen[record.ID]; ok {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.id", "template ids must be unique")
		}
		seen[record.ID] = struct{}{}
		if record.Order <= 0 {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.order", "template order must be positive")
		}
		if record.Template.URL == "" {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.template.url", "template URL is required")
		}
		if record.Hash != "" && !IsValidSHA256Hash(record.Hash) {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.hash", "hash must be a valid SHA256 string")
		}
		vars, ok := varsByID[record.ID]
		if !ok {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates", "missing variables record for template "+record.ID)
		}
		if vars == nil {
			vars = map[string]interface{}{}
		}
		files, ok := filesByID[record.ID]
		if !ok {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates", "missing manifest record for template "+record.ID)
		}
		if manifestOrder := orderByManifestID[record.ID]; manifestOrder != record.Order {
			return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.order", "manifest order must match ign.json order")
		}
		templates = append(templates, ProjectTemplateState{
			ID:        record.ID,
			Order:     record.Order,
			Template:  record.Template,
			Hash:      record.Hash,
			Metadata:  record.Metadata,
			Variables: vars,
			Files:     files,
		})
	}
	if len(varsByID) != len(templates) || len(filesByID) != len(templates) {
		return nil, NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates", "metadata template records must match across files")
	}

	sort.SliceStable(templates, func(i, j int) bool {
		return templates[i].Order < templates[j].Order
	})
	return &ProjectState{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: outputRoot,
		Templates:  templates,
	}, nil
}

func validateProjectStateForSave(configDir string, outputRoot string, templates []ProjectTemplateState) error {
	if err := validateProjectStateOutputRoot(configDir, outputRoot); err != nil {
		return err
	}
	if len(templates) == 0 {
		return NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates", "at least one template record is required")
	}
	seen := make(map[string]struct{}, len(templates))
	for _, tmpl := range templates {
		if tmpl.ID == "" {
			return NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.id", "template id is required")
		}
		if _, ok := seen[tmpl.ID]; ok {
			return NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.id", "template ids must be unique")
		}
		seen[tmpl.ID] = struct{}{}
		if tmpl.Order <= 0 {
			return NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.order", "template order must be positive")
		}
		if tmpl.Template.URL == "" {
			return NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.template.url", "template URL is required")
		}
		if tmpl.Hash != "" && !IsValidSHA256Hash(tmpl.Hash) {
			return NewConfigErrorWithField(ConfigValidationFailed, configDir, "templates.hash", "hash must be a valid SHA256 string")
		}
	}
	return nil
}

func validateProjectStateOutputRoot(configDir string, outputRoot string) error {
	if outputRoot == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, configDir, "output_root", "output_root is required")
	}
	clean := filepath.Clean(outputRoot)
	if clean == "" {
		clean = "."
	}
	if clean != outputRoot || filepath.IsAbs(outputRoot) || hasTraversalComponent(clean) {
		return NewConfigErrorWithField(ConfigValidationFailed, configDir, "output_root", "output_root must be a normalized relative path")
	}
	return nil
}

func hasTraversalComponent(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func writeJSONAtomic(path string, value interface{}) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return NewConfigErrorWithCause(ConfigInvalid, path, "path contains '..' which is not allowed for security reasons", nil)
	}
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, fmt.Sprintf("failed to create directory %s", dir), err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to marshal JSON", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(cleanPath)+".tmp-*")
	if err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to create temporary file", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to write temporary file", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to close temporary file", err)
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil {
		_ = os.Remove(tmpPath)
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to replace file", err)
	}
	return nil
}

func normalizeOutputRoot(outputRoot string) string {
	if outputRoot == "" {
		return ""
	}
	clean := filepath.Clean(outputRoot)
	if clean == "" {
		return "."
	}
	return clean
}

func cleanManifestFiles(files []string) []string {
	seen := make(map[string]struct{}, len(files))
	result := make([]string, 0, len(files))
	for _, path := range files {
		clean := filepath.Clean(path)
		if clean == "" || clean == "." {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	sort.Strings(result)
	return result
}

func templateRecordIDFromMetadata(metadata *model.FileMetadata, order int) string {
	base := "template"
	if metadata != nil && metadata.TemplateName != "" {
		base = metadata.TemplateName
	}
	return fmt.Sprintf("%s-%d", sanitizeTemplateIDBase(base), order)
}

func sanitizeTemplateIDBase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "template"
	}
	return result
}
