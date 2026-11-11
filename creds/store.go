// Package creds checks the credentials of the configured platforms and writes them back into the
// configuration file.
//
// Everything here is non-interactive, on purpose : the CLI commands and the full screen TUI both
// call it, and they cannot share their prompts. Each of them asks its own questions and calls these
// functions with the answers.
package creds

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// The fields of a platform which can be written in the configuration file
const (
	FieldToken      = "token"
	FieldType       = "type"
	FieldPublicKey  = "public_key"
	FieldPassphrase = "public_key_passphrase"
)

// ErrUnknownField is returned for a field which is not one of the constants above
var ErrUnknownField = errors.New("unknown platform field")

var writableFields = map[string]bool{
	FieldToken:      true,
	FieldType:       true,
	FieldPublicKey:  true,
	FieldPassphrase: true,
}

// SetPlatformField writes one field of one platform into the configuration file
func SetPlatformField(path, host, field, value string) error {
	return SetPlatformFields(path, []string{host}, field, value)
}

// SetPlatformFields writes the same value in the same field of several platforms, in a single read /
// change / save of the configuration file, so that a bulk update is all or nothing.
//
// Comments of the file are kept, the file is created when it does not exist, and it is replaced
// atomically with mode 0600 : it may hold a token.
func SetPlatformFields(path string, hosts []string, field, value string) error {
	if !writableFields[field] {
		return fmt.Errorf("%w : %s", ErrUnknownField, field)
	}
	if len(hosts) == 0 {
		return errors.New("no platform to update")
	}

	doc, err := load(path)
	if err != nil {
		return err
	}
	root := doc.Content[0]
	platforms := ensureMapping(root, "platforms")
	for _, host := range hosts {
		if strings.TrimSpace(host) == "" {
			return errors.New("the host of a platform cannot be empty")
		}
		setScalar(ensureMapping(platforms, host), field, value)
	}
	return save(path, doc)
}

// EnsureWritable tells whether the configuration file can be replaced, so that the user is not asked
// for a secret which could not be saved afterwards
func EnsureWritable(path string) error {
	path = resolve(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create %s : %w", dir, err)
	}
	// The file is replaced by a rename, so it is the directory which has to be writable
	probe, err := os.CreateTemp(dir, ".heimdall-probe-*")
	if err != nil {
		return fmt.Errorf("cannot write in %s : %w", dir, err)
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)

	// A file the user made read-only would still be replaced by the rename, so refuse it explicitly
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", path)
		}
		if info.Mode().Perm()&0200 == 0 {
			return fmt.Errorf("%s is read-only", path)
		}
	}
	return nil
}

// load reads the configuration file as a yaml tree, keeping its comments. A missing or empty file
// gives an empty document rather than an error, so that the first credential can be written.
func load(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return emptyDocument(), nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("cannot read %s : %w", path, err)
	}
	// An empty file, or one holding only comments, decodes into a node with no content
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return emptyDocument(), nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s does not hold a yaml mapping", path)
	}
	return &doc, nil
}

func emptyDocument() *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
	}
}

// save writes the tree back, with the two space indentation of the sample configuration file, into a
// temporary file of the same directory which then replaces the configuration file
func save(path string, doc *yaml.Node) error {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	path = resolve(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// os.CreateTemp already creates it readable by its owner only, which matters : it holds a copy of
	// the whole configuration file for as long as the rename takes
	tmp, err := os.CreateTemp(dir, ".heimdall-*.yml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Whatever happens below, no copy of the configuration is left behind
	renamed := false
	defer func() {
		if !renamed {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(buffer.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	// Without this the rename can be durable before the content is, which would leave an empty
	// configuration file after a crash : every configured platform would be lost
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	renamed = true
	return nil
}

// resolve follows a symbolic link to the configuration file, so that a file linked from a dotfiles
// repository is updated where it really lives instead of being replaced by a regular file
func resolve(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

// find returns the value node of a key of a mapping, and the key node itself. Keys are compared
// without case, because the configuration is read through viper, which lowercases them : looking for
// "gitlab.com" has to find a "GitLab.com" already in the file instead of adding a second entry.
func find(mapping *yaml.Node, key string) (keyNode, valueNode *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if strings.EqualFold(mapping.Content[i].Value, key) {
			return mapping.Content[i], mapping.Content[i+1]
		}
	}
	return nil, nil
}

// ensureMapping returns the mapping held under a key, adding it when it is missing. A key holding
// something else than a mapping is replaced, as it cannot be a platform.
func ensureMapping(mapping *yaml.Node, key string) *yaml.Node {
	if keyNode, valueNode := find(mapping, key); keyNode != nil {
		if valueNode.Kind == yaml.MappingNode {
			return valueNode
		}
		*valueNode = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return valueNode
	}
	added := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		added)
	return added
}

// setScalar writes a scalar value under a key. An existing entry keeps its comments and its place in
// the file : only its value changes.
func setScalar(mapping *yaml.Node, key, value string) {
	if _, valueNode := find(mapping, key); valueNode != nil {
		valueNode.Kind = yaml.ScalarNode
		valueNode.Tag = "!!str"
		valueNode.Value = value
		valueNode.Content = nil
		// A value which was quoted, or written on several lines, is rewritten plainly
		valueNode.Style = 0
		return
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
}
