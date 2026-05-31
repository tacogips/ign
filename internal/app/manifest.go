package app

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func manifestPath() string {
	return filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
}

func manifestPathFromConfigPath(configPath string) string {
	if configPath == "" {
		return manifestPath()
	}
	return filepath.Join(filepath.Dir(configPath), model.IgnManifestFile)
}

func backupManifestIfExists() error {
	path := manifestPath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	backupNum, err := findNextBackupNumber(model.IgnConfigDir, model.IgnManifestFile)
	if err != nil {
		return err
	}

	backupPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile+".bk"+strconv.Itoa(backupNum))
	return os.Rename(path, backupPath)
}

func saveManifestFromGenerateResult(path string, result *generator.GenerateResult) error {
	return saveManifestFromGenerateResultExcluding(path, result, nil)
}

func saveManifestFromGenerateResultExcluding(path string, result *generator.GenerateResult, excludedCanonicalPaths map[string]struct{}) error {
	if result == nil {
		return nil
	}

	manifest, err := loadManifestOrEmpty(path)
	if err != nil {
		return err
	}

	files := make([]string, 0, len(manifest.Files)+len(result.WrittenFiles)+len(result.CreatedFiles))
	seen := make(map[string]struct{}, len(manifest.Files))
	for _, manifestFile := range manifest.Files {
		if isExcludedManifestPath(manifestFile, excludedCanonicalPaths) {
			continue
		}
		clean := filepath.Clean(manifestFile)
		if clean == "" || clean == "." {
			continue
		}
		seen[clean] = struct{}{}
		files = append(files, clean)
	}

	writtenPaths := result.WrittenFiles
	if len(writtenPaths) == 0 {
		writtenPaths = result.CreatedFiles
	}

	for _, path := range writtenPaths {
		clean := filepath.Clean(path)
		if clean == "" || clean == "." {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		files = append(files, clean)
		seen[clean] = struct{}{}
	}

	manifest.Files = files
	sort.Strings(manifest.Files)
	return config.SaveIgnManifest(path, manifest)
}

func isExcludedManifestPath(path string, excludedCanonicalPaths map[string]struct{}) bool {
	if len(excludedCanonicalPaths) == 0 {
		return false
	}
	canonical, err := canonicalManagedPathForComparison(path)
	if err != nil {
		return false
	}
	_, ok := excludedCanonicalPaths[canonical]
	return ok
}

func loadManifestOrEmpty(path string) (*model.IgnManifest, error) {
	manifest, err := config.LoadIgnManifest(path)
	if err == nil {
		return manifest, nil
	}
	if cfgErr, ok := err.(*config.ConfigError); ok && cfgErr.Type == config.ConfigNotFound {
		return &model.IgnManifest{Files: []string{}}, nil
	}
	return nil, err
}
