//go:build darwin || linux

package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

// symlinkTransitionSnapshotNode identifies one node that belonged to the
// verified source tree. It is persisted with the journal for source-state and
// recovery validation; committed backups are archived rather than deleted.
type symlinkTransitionSnapshotNode struct {
	Path   string `json:"path"`
	Mode   uint32 `json:"mode"`
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
	Target string `json:"target,omitempty"`
	Digest string `json:"digest,omitempty"`
}

// fingerprintTransitionSource captures a candidate directory through
// descriptor-relative, no-follow operations. Its result is compared after the
// directory has been renamed to the transaction backup, closing the interval
// between execution-plan validation and replacement.
func fingerprintTransitionSource(outputDir, path string) (string, error) {
	parentFD, name, err := openTransitionParent(outputDir, path)
	if err != nil {
		return "", err
	}
	defer func() { _ = unix.Close(parentFD) }()
	fingerprint, _, err := captureTransitionTreeAt(parentFD, name)
	return fingerprint, err
}

func fingerprintTransitionTreeAt(parentFD int, name string) (string, error) {
	fingerprint, _, err := captureTransitionTreeAt(parentFD, name)
	return fingerprint, err
}

// captureTransitionTreeAt records a single descriptor-relative traversal.
// Capturing the fingerprint and journal snapshot together records the exact
// tree authorized for replacement without following any descendant symlink.
func captureTransitionTreeAt(parentFD int, name string) (string, []symlinkTransitionSnapshotNode, error) {
	h := sha256.New()
	nodes := make([]symlinkTransitionSnapshotNode, 0)
	if err := captureTransitionNodeAt(h, parentFD, name, ".", &nodes); err != nil {
		return "", nil, err
	}
	return hex.EncodeToString(h.Sum(nil)), nodes, nil
}

func fingerprintTransitionNodeAt(h io.Writer, parentFD int, name string) error {
	return captureTransitionNodeAt(h, parentFD, name, ".", nil)
}

func captureTransitionNodeAt(h io.Writer, parentFD int, name, relativePath string, nodes *[]symlinkTransitionSnapshotNode) error {
	if name == "" || name == "." || name == ".." || strings.Contains(name, string(filepath.Separator)) {
		return fmt.Errorf("invalid transition tree entry name")
	}
	var stat unix.Stat_t
	if err := unix.Fstatat(parentFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return fmt.Errorf("inspect transition tree entry: %w", err)
	}
	if err := writeFingerprintField(h, "transition-mode", []byte(fmt.Sprintf("%#o", stat.Mode))); err != nil {
		return err
	}
	if err := writeFingerprintField(h, "transition-device", []byte(fmt.Sprintf("%d", stat.Dev))); err != nil {
		return err
	}
	if err := writeFingerprintField(h, "transition-inode", []byte(fmt.Sprintf("%d", stat.Ino))); err != nil {
		return err
	}
	node := symlinkTransitionSnapshotNode{
		Path:   relativePath,
		Mode:   uint32(stat.Mode & unix.S_IFMT),
		Device: uint64(stat.Dev),
		Inode:  uint64(stat.Ino),
	}
	switch stat.Mode & unix.S_IFMT {
	case unix.S_IFLNK:
		target, err := readlinkAt(parentFD, name)
		if err != nil {
			return fmt.Errorf("read transition tree symlink: %w", err)
		}
		node.Target = target
		if nodes != nil {
			*nodes = append(*nodes, node)
		}
		return writeFingerprintField(h, "transition-symlink-target", []byte(target))
	case unix.S_IFDIR:
		if nodes != nil {
			*nodes = append(*nodes, node)
		}
		fd, err := unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return fmt.Errorf("open transition tree directory: %w", err)
		}
		dir := os.NewFile(uintptr(fd), name)
		entries, readErr := dir.ReadDir(-1)
		if readErr != nil {
			_ = dir.Close()
			return fmt.Errorf("read transition tree directory: %w", readErr)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if err := writeFingerprintField(h, "transition-directory-entry", []byte(entry.Name())); err != nil {
				_ = dir.Close()
				return err
			}
			childPath := filepath.Join(relativePath, entry.Name())
			if err := captureTransitionNodeAt(h, fd, entry.Name(), childPath, nodes); err != nil {
				_ = dir.Close()
				return err
			}
		}
		if err := dir.Close(); err != nil {
			return fmt.Errorf("close transition tree directory: %w", err)
		}
		return writeFingerprintField(h, "transition-directory-end", nil)
	case unix.S_IFREG:
		fd, err := unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return fmt.Errorf("open transition tree file: %w", err)
		}
		file := os.NewFile(uintptr(fd), name)
		contents, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return fmt.Errorf("read transition tree file: %w", readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close transition tree file: %w", closeErr)
		}
		digest := sha256.Sum256(contents)
		node.Digest = hex.EncodeToString(digest[:])
		if nodes != nil {
			*nodes = append(*nodes, node)
		}
		return writeFingerprintField(h, "transition-file-content", contents)
	default:
		return fmt.Errorf("unsupported transition tree entry type")
	}
}

func validateTransitionSnapshot(nodes []symlinkTransitionSnapshotNode) error {
	if len(nodes) == 0 {
		return fmt.Errorf("missing transition cleanup snapshot")
	}
	paths := make(map[string]struct{}, len(nodes))
	hasRoot := false
	for _, node := range nodes {
		if err := validateTransitionSnapshotPath(node.Path); err != nil {
			return err
		}
		if node.Mode != unix.S_IFDIR && node.Mode != unix.S_IFREG && node.Mode != unix.S_IFLNK {
			return fmt.Errorf("invalid transition snapshot node type")
		}
		if node.Mode == unix.S_IFLNK && node.Target == "" {
			return fmt.Errorf("missing transition snapshot symlink target")
		}
		if node.Mode == unix.S_IFREG {
			if len(node.Digest) != sha256.Size*2 {
				return fmt.Errorf("invalid transition snapshot file digest")
			}
			if _, err := hex.DecodeString(node.Digest); err != nil {
				return fmt.Errorf("invalid transition snapshot file digest: %w", err)
			}
		}
		if _, exists := paths[node.Path]; exists {
			return fmt.Errorf("duplicate transition snapshot path")
		}
		paths[node.Path] = struct{}{}
		if node.Path == "." {
			if node.Mode != unix.S_IFDIR {
				return fmt.Errorf("transition snapshot root is not a directory")
			}
			hasRoot = true
		}
	}
	if !hasRoot {
		return fmt.Errorf("missing transition snapshot root")
	}
	return nil
}

// validateTransitionSnapshotOwnership proves that the descriptor-relative tree
// renamed into the transaction backup still contains only the manifest-owned
// terminal nodes authorized by the execution plan. It closes the interval
// between preview-time classification and the source fingerprint capture.
func validateTransitionSnapshotOwnership(root string, nodes []symlinkTransitionSnapshotNode, retiredManagedPaths []string) error {
	managed := make(map[string]struct{}, len(retiredManagedPaths))
	for _, path := range retiredManagedPaths {
		managed[filepath.Clean(path)] = struct{}{}
	}
	children := make(map[string]int, len(nodes))
	for _, node := range nodes {
		if node.Path != "." {
			children[filepath.Dir(node.Path)]++
		}
	}
	for _, node := range nodes {
		switch node.Mode {
		case unix.S_IFDIR:
			if children[node.Path] == 0 {
				return fmt.Errorf("renamed managed directory contains an empty subtree %q", node.Path)
			}
		case unix.S_IFREG, unix.S_IFLNK:
			path := filepath.Clean(filepath.Join(root, node.Path))
			if _, ok := managed[path]; !ok {
				return fmt.Errorf("renamed managed directory contains untracked entry %q", node.Path)
			}
		default:
			return fmt.Errorf("renamed managed directory contains unsupported entry type")
		}
	}
	return nil
}

func validateTransitionSnapshotPath(path string) error {
	if path == "." {
		return nil
	}
	if filepath.IsAbs(path) || path == "" || strings.HasPrefix(path, ".."+string(filepath.Separator)) || path == ".." {
		return fmt.Errorf("invalid transition snapshot path")
	}
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid transition snapshot path component")
		}
	}
	return nil
}

func unusedBackupName(parentFD int) (string, error) {
	for range 32 {
		bytes := make([]byte, 12)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate transaction backup name: %w", err)
		}
		name := ".ign-symlink-transition-" + hex.EncodeToString(bytes)
		var stat unix.Stat_t
		err := unix.Fstatat(parentFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
		if err == unix.ENOENT {
			return name, nil
		}
		if err != nil {
			return "", fmt.Errorf("check transaction backup path: %w", err)
		}
	}
	return "", fmt.Errorf("allocate unique transaction backup name")
}
