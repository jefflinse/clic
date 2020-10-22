package registry

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const registryFileName = ".handyman_registry"

// A Registry is a map of app names to spec file paths.
type Registry map[string]string

// Load loads the registry from disk.
func Load() (Registry, error) {
	file, err := registryFilePath()
	if err != nil {
		return nil, err
	}

	if !fileExists(file) {
		if err := createRegistryFile(file); err != nil {
			return nil, err
		}
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	reg := Registry{}
	if len(content) == 0 {
		return reg, nil
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid registry entry (line %d): '%s'", i, line)
		}

		name := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])
		if name == "" || path == "" {
			return nil, fmt.Errorf("invalid registry entry (line %d): '%s'", i, line)
		}

		reg[name] = path
	}

	return reg, nil
}

func (r Registry) Add(name string, path string) error {
	if path, ok := r[name]; ok {
		return fmt.Errorf("'%s' already registered as '%s'", name, path)
	}

	r[name] = path

	return r.Save()
}

func (r Registry) Remove(name string) error {
	if _, ok := r[name]; !ok {
		return fmt.Errorf("'%s' is not registered", name)
	}

	delete(r, name)

	return r.Save()
}

func (r Registry) Save() error {
	builder := strings.Builder{}
	for name, path := range r {
		builder.WriteString(fmt.Sprintf("%s = %s", name, path))
	}

	file, err := registryFilePath()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(file, []byte(builder.String()), 0644)
}

func createRegistryFile(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	return file.Close()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func registryFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homeDir, registryFileName), nil
}
