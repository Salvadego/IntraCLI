package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	cacheDirName               = ".cache"
	appName                    = "intracli"
	TimesheetsCacheFileName    = "timesheets%d.json"
	EmployeeListCacheFileName  = "employeesList.json"
	ContractsListCacheFileName = "contractsList.json"
)

func GetCacheFilePath(cacheFileName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	cacheDirPath := filepath.Join(homeDir, cacheDirName, appName)
	return filepath.Join(cacheDirPath, cacheFileName), nil
}

func EnsureCacheDirExists() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}
	cacheDirPath := filepath.Join(homeDir, cacheDirName, appName)
	return os.MkdirAll(cacheDirPath, 0755)
}

func WriteToCache[T any](cacheFileName string, data []T) error {
	if err := EnsureCacheDirExists(); err != nil {
		return fmt.Errorf("failed to ensure cache directory exists: %w", err)
	}

	cacheFilePath, err := GetCacheFilePath(cacheFileName)
	if err != nil {
		return fmt.Errorf("failed to get cache file path: %w", err)
	}

	marshalData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal list to JSON: %w", err)
	}

	if err := os.WriteFile(cacheFilePath, marshalData, 0644); err != nil {
		return fmt.Errorf("failed to write list to cache file: %w", err)
	}
	return nil
}

func ReadFromCache[T any](cacheFileName string) ([]T, error) {
	cacheFilePath, err := GetCacheFilePath(cacheFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache file path: %w", err)
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []T{}, fmt.Errorf("Cache file not found at %s. Returning empty list.\n", cacheFilePath)
		}
		return nil, fmt.Errorf("failed to read list from cache file: %w", err)
	}

	var unData []T
	if err := json.Unmarshal(data, &unData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal  from JSON: %w", err)
	}
	return unData, nil
}
