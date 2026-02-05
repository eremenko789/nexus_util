package asset

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"nexus-util/config"
	"nexus-util/nexus"

	"github.com/spf13/cobra"
)

var DiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare assets between repositories or repository and local directory",
	Long: `Compare files by presence and checksum.
The command can compare:
  - Two Nexus repositories (possibly on different servers)
  - One Nexus repository and a local directory

Output is always JSON with file lists grouped by comparison result.

Examples:
  # Compare two repositories on different servers
  nexus-util asset diff -a http://source.example.com -r repo1 \
    --target-address http://target.example.com --target-repo repo2

  # Compare a repository subpath against a local directory
  nexus-util asset diff -a http://nexus.example.com -r repo1 \
    --path releases/v1.2.3 --local ./downloads

  # Compare excluding a specific subdirectory
  nexus-util asset diff -a http://nexus.example.com -r repo1 \
    --path releases/v1.2.3 --local ./downloads --exclude releases/v1.2.3/temp
`,
	Args: cobra.NoArgs,
	RunE: runDiff,
}

type diffResult struct {
	Identical  []diffFile     `json:"identical"`
	OnlySource []string       `json:"only_source"`
	OnlyTarget []string       `json:"only_target"`
	Different  []diffMismatch `json:"different"`
}

type diffFile struct {
	Path      string `json:"path"`
	Algorithm string `json:"algorithm,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

type diffMismatch struct {
	Path       string `json:"path"`
	Algorithm  string `json:"algorithm,omitempty"`
	SourceHash string `json:"source_hash,omitempty"`
	TargetHash string `json:"target_hash,omitempty"`
}

type fileEntry struct {
	RelativePath string
	Asset        *nexus.Asset
	LocalPath    string
}

var hashPreference = []string{"sha256", "sha1", "md5"}

func runDiff(cmd *cobra.Command, _ []string) error {
	// Source (default) flags
	address, _ := cmd.Flags().GetString("address")
	repository, _ := cmd.Flags().GetString("repository")
	username, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	configPath, _ := cmd.Flags().GetString("config")
	dryRun, _ := cmd.Flags().GetBool("dry")
	insecure, _ := cmd.Flags().GetBool("insecure")

	// Target/local flags
	targetAddress, _ := cmd.Flags().GetString("target-address")
	targetRepo, _ := cmd.Flags().GetString("target-repo")
	targetUser, _ := cmd.Flags().GetString("target-user")
	targetPass, _ := cmd.Flags().GetString("target-pass")
	localDir, _ := cmd.Flags().GetString("local")
	pathFlag, _ := cmd.Flags().GetString("path")
	excludeDir, _ := cmd.Flags().GetString("exclude")

	// Definening the work scenario
	var scenario string
	if localDir != "" {
		scenario = "local"
	} else if targetAddress != "" || targetRepo != "" {
		scenario = "nexus-to-nexus"
	} else {
		return fmt.Errorf("must specify either --local or --target-* flags")
	}

	// Checking for conflicting flags
	if scenario == "local" && (targetAddress != "" || targetRepo != "" || targetUser != "" || targetPass != "") {
		return fmt.Errorf("use either --local or --target-* flags, not both")
	}

	// Load source config
	cfg, err := config.LoadConfigWithFlags(configPath, map[string]interface{}{
		"nexusAddress": address,
		"user":         username,
		"password":     password,
	})
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	sourceAddress := address
	if sourceAddress == "" {
		sourceAddress = cfg.GetNexusAddress()
	}
	if sourceAddress == "" {
		return fmt.Errorf("source address is required (use --address or config)")
	}
	if !strings.HasPrefix(sourceAddress, "http://") && !strings.HasPrefix(sourceAddress, "https://") {
		return fmt.Errorf("source address must include protocol (http:// or https://)")
	}

	sourceUser := username
	if sourceUser == "" {
		sourceUser = cfg.GetUser()
	}
	sourcePass := password
	if sourcePass == "" {
		sourcePass = cfg.GetPassword()
	}

	if repository == "" {
		return fmt.Errorf("source repository is required")
	}

	normalizedPath := "/" + normalizeRepoPath(pathFlag)
	normalizedExclude := "/" + normalizeRepoPath(excludeDir)

	// Always silence Nexus client logs to keep JSON clean.
	sourceClient := nexus.NewNexusClient(sourceAddress, sourceUser, sourcePass, true, dryRun, insecure)

	var sourceFiles map[string]fileEntry
	sourceFiles, err = collectRepoFiles(sourceClient, repository, normalizedPath)
	if err != nil {
		return fmt.Errorf("failed to load source repository files: %w", err)
	}

	// Applying an exclusion to source files
	if normalizedExclude != "" {
		sourceFiles = filterExcludedFiles(sourceFiles, normalizedExclude)
	}

	var targetFiles map[string]fileEntry
	var targetClient *nexus.NexusClient

	switch scenario {
	case "local":
		localRoot := localDir
		if normalizedPath != "" {
			localRoot = filepath.Join(localDir, filepath.FromSlash(normalizedPath))
		}
		targetFiles, err = collectLocalFiles(localRoot)
		if err != nil {
			return fmt.Errorf("failed to load local files: %w", err)
		}

		// Applying an exclusion to local files
		if normalizedExclude != "" {
			targetFiles = filterExcludedFiles(targetFiles, normalizedExclude)
		}

	case "nexus-to-nexus":
		// Setting up the target client
		if targetAddress == "" {
			targetAddress = sourceAddress
		}
		if targetUser == "" {
			targetUser = sourceUser
		}
		if targetPass == "" {
			targetPass = cfg.GetPassword()
		}

		// Check that the target repository is specified
		if targetRepo == "" {
			return fmt.Errorf("target repository is required when comparing repositories")
		}

		targetClient = nexus.NewNexusClient(targetAddress, targetUser, targetPass, true, dryRun, insecure)
		targetFiles, err = collectRepoFiles(targetClient, targetRepo, normalizedPath)
		if err != nil {
			return fmt.Errorf("failed to load target repository files: %w", err)
		}

		// Applying an exclusion to target files
		if normalizedExclude != "" {
			targetFiles = filterExcludedFiles(targetFiles, normalizedExclude)
		}

	}

	result := diffResult{
		Identical:  []diffFile{},
		OnlySource: []string{},
		OnlyTarget: []string{},
		Different:  []diffMismatch{},
	}

	for relPath, sourceEntry := range sourceFiles {
		targetEntry, ok := targetFiles[relPath]
		if !ok {
			result.OnlySource = append(result.OnlySource, relPath)
			continue
		}

		algorithm, sourceHash, targetHash, err := comparableHashes(sourceEntry, targetEntry, sourceClient, targetClient)
		if err != nil {
			return fmt.Errorf("failed to compare '%s': %w", relPath, err)
		}

		if strings.EqualFold(sourceHash, targetHash) {
			result.Identical = append(result.Identical, diffFile{
				Path:      relPath,
				Algorithm: algorithm,
				Hash:      strings.ToLower(sourceHash),
			})
		} else {
			result.Different = append(result.Different, diffMismatch{
				Path:       relPath,
				Algorithm:  algorithm,
				SourceHash: strings.ToLower(sourceHash),
				TargetHash: strings.ToLower(targetHash),
			})
		}
	}

	for relPath := range targetFiles {
		if _, ok := sourceFiles[relPath]; !ok {
			result.OnlyTarget = append(result.OnlyTarget, relPath)
		}
	}

	sort.Strings(result.OnlySource)
	sort.Strings(result.OnlyTarget)
	sort.Slice(result.Identical, func(i, j int) bool {
		return result.Identical[i].Path < result.Identical[j].Path
	})
	sort.Slice(result.Different, func(i, j int) bool {
		return result.Different[i].Path < result.Different[j].Path
	})

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func normalizeRepoPath(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	return strings.Trim(trimmed, "/")
}

func relativeAssetPath(assetPath string, root string) string {
	root = normalizeRepoPath(root)
	assetPath = strings.ReplaceAll(assetPath, "\\", "/")
	if root == "" {
		return assetPath
	}
	if assetPath == root {
		return path.Base(assetPath)
	}
	prefix := root + "/"
	if strings.HasPrefix(assetPath, prefix) {
		return strings.TrimPrefix(assetPath, prefix)
	}
	return assetPath
}

func collectRepoFiles(client *nexus.NexusClient, repository string, root string) (map[string]fileEntry, error) {
	assets, err := client.GetFilesInDirectory(repository, root)
	if err != nil {
		return nil, err
	}

	files := make(map[string]fileEntry, len(assets))
	for _, asset := range assets {
		relPath := filepath.ToSlash(relativeAssetPath(asset.Path, root))
		assetCopy := asset
		files[relPath] = fileEntry{
			RelativePath: relPath,
			Asset:        &assetCopy,
		}
	}
	return files, nil
}

func collectLocalFiles(root string) (map[string]fileEntry, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	files := make(map[string]fileEntry)
	if !info.IsDir() {
		relPath := filepath.ToSlash(filepath.Base(root))
		files[relPath] = fileEntry{
			RelativePath: relPath,
			LocalPath:    root,
		}
		return files, nil
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		files[relPath] = fileEntry{
			RelativePath: relPath,
			LocalPath:    path,
		}
		return nil
	})

	return files, err
}

func normalizeChecksumMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		if value == "" {
			continue
		}
		output[strings.ToLower(key)] = strings.ToLower(value)
	}
	return output
}

func comparableHashes(source fileEntry, target fileEntry, sourceClient *nexus.NexusClient, targetClient *nexus.NexusClient) (string, string, string, error) {
	sourceHashes := map[string]string{}
	if source.Asset != nil && source.Asset.Checksum != nil {
		sourceHashes = normalizeChecksumMap(source.Asset.Checksum)
	}

	targetHashes := map[string]string{}
	if target.Asset != nil && target.Asset.Checksum != nil {
		targetHashes = normalizeChecksumMap(target.Asset.Checksum)
	}

	for _, algorithm := range hashPreference {
		if sourceHashes[algorithm] != "" && targetHashes[algorithm] != "" {
			return algorithm, sourceHashes[algorithm], targetHashes[algorithm], nil
		}
	}

	chosen := ""
	for _, algorithm := range hashPreference {
		if sourceHashes[algorithm] != "" || targetHashes[algorithm] != "" {
			chosen = algorithm
			break
		}
	}
	if chosen == "" {
		chosen = hashPreference[0]
	}

	if sourceHashes[chosen] == "" {
		hashValue, err := computeHashForEntry(source, sourceClient, chosen)
		if err != nil {
			return "", "", "", err
		}
		sourceHashes[chosen] = strings.ToLower(hashValue)
	}

	if targetHashes[chosen] == "" {
		hashValue, err := computeHashForEntry(target, targetClient, chosen)
		if err != nil {
			return "", "", "", err
		}
		targetHashes[chosen] = strings.ToLower(hashValue)
	}

	return chosen, sourceHashes[chosen], targetHashes[chosen], nil
}

func computeHashForEntry(entry fileEntry, client *nexus.NexusClient, algorithm string) (string, error) {
	if entry.Asset != nil {
		if entry.Asset.DownloadUrl == "" {
			return "", fmt.Errorf("download URL missing for %s", entry.Asset.Path)
		}
		if client == nil {
			return "", fmt.Errorf("nexus client is required to hash %s", entry.Asset.Path)
		}
		return client.ComputeHashFromDownloadURL(entry.Asset.DownloadUrl, algorithm)
	}
	if entry.LocalPath == "" {
		return "", fmt.Errorf("local path is missing for %s", entry.RelativePath)
	}
	return computeLocalHash(entry.LocalPath, algorithm)
}

func computeLocalHash(filePath string, algorithm string) (string, error) {
	hasher, err := newLocalHashForAlgorithm(algorithm)
	if err != nil {
		return "", err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func filterExcludedFiles(files map[string]fileEntry, excludePath string) map[string]fileEntry {
	filtered := make(map[string]fileEntry)
	excludePrefix := excludePath + "/"

	for path, entry := range files {
		if strings.HasPrefix(path, excludePrefix) {
			continue
		}
		filtered[path] = entry
	}

	return filtered
}

func newLocalHashForAlgorithm(algorithm string) (hash.Hash, error) {
	switch strings.ToLower(algorithm) {
	case "sha256":
		return sha256.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "md5":
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}
