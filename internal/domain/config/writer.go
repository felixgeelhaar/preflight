package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PatchOp indicates the type of patch operation.
type PatchOp string

// PatchOp constants.
const (
	PatchOpAdd    PatchOp = "add"
	PatchOpModify PatchOp = "modify"
	PatchOpRemove PatchOp = "remove"
)

// Patch represents a change to be made to a layer YAML file.
type Patch struct {
	LayerPath  string
	YAMLPath   string
	Operation  PatchOp
	OldValue   interface{}
	NewValue   interface{}
	Provenance string
}

// LayerWriter applies patches to layer YAML files while preserving comments.
type LayerWriter struct{}

// NewLayerWriter creates a new LayerWriter.
func NewLayerWriter() *LayerWriter {
	return &LayerWriter{}
}

// ApplyPatch applies a single patch to a layer file.
func (w *LayerWriter) ApplyPatch(patch Patch) error {
	// Read the file
	data, err := os.ReadFile(patch.LayerPath)
	if err != nil {
		return fmt.Errorf("failed to read layer file: %w", err)
	}

	// Parse YAML preserving comments
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Parse the path
	pathParts := parsePath(patch.YAMLPath)

	// Apply the patch based on operation
	switch patch.Operation {
	case PatchOpAdd:
		if err := w.applyAdd(&root, pathParts, patch.NewValue); err != nil {
			return err
		}
	case PatchOpModify:
		if err := w.applyModify(&root, pathParts, patch.NewValue); err != nil {
			return err
		}
	case PatchOpRemove:
		if err := w.applyRemove(&root, pathParts); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown patch operation: %s", patch.Operation)
	}

	// Write back the file
	output, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(patch.LayerPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write layer file: %w", err)
	}

	return nil
}

// ApplyPatches applies multiple patches to layer files.
func (w *LayerWriter) ApplyPatches(patches []Patch) error {
	for _, patch := range patches {
		if err := w.ApplyPatch(patch); err != nil {
			return err
		}
	}
	return nil
}

// parsePath parses a YAML path like "parent.child[0].key" into parts.
func parsePath(path string) []pathPart {
	var parts []pathPart

	// Split by dots, handling array indices
	re := regexp.MustCompile(`([^.\[\]]+)|\[(\d+)\]`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if match[1] != "" {
			parts = append(parts, pathPart{key: match[1]})
		} else if match[2] != "" {
			idx, _ := strconv.Atoi(match[2])
			parts = append(parts, pathPart{index: idx, isIndex: true})
		}
	}

	return parts
}

type pathPart struct {
	key     string
	index   int
	isIndex bool
}

func (w *LayerWriter) applyAdd(root *yaml.Node, parts []pathPart, value interface{}) error {
	node, parent := w.findNode(root, parts)
	if node != nil && !parts[len(parts)-1].isIndex {
		// Key exists, treat as modify
		return w.applyModify(root, parts, value)
	}

	// Create value node
	valueNode := &yaml.Node{}
	valueBytes, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(valueBytes, valueNode); err != nil {
		return err
	}

	if parent == nil {
		// Adding to root
		if root.Kind == 0 {
			root.Kind = yaml.DocumentNode
			root.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
		}
		parent = root.Content[0]
	}

	lastPart := parts[len(parts)-1]
	if lastPart.isIndex {
		// Adding to array
		if parent.Kind != yaml.SequenceNode {
			return fmt.Errorf("cannot add index to non-sequence node")
		}
		if valueNode.Kind == yaml.DocumentNode {
			parent.Content = append(parent.Content, valueNode.Content[0])
		} else {
			parent.Content = append(parent.Content, valueNode)
		}
	} else {
		// Adding to map
		if parent.Kind != yaml.MappingNode {
			return fmt.Errorf("cannot add key to non-mapping node")
		}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: lastPart.key}
		if valueNode.Kind == yaml.DocumentNode {
			parent.Content = append(parent.Content, keyNode, valueNode.Content[0])
		} else {
			parent.Content = append(parent.Content, keyNode, valueNode)
		}
	}

	return nil
}

func (w *LayerWriter) applyModify(root *yaml.Node, parts []pathPart, value interface{}) error {
	node, _ := w.findNode(root, parts)
	if node == nil {
		return fmt.Errorf("path not found: %s", pathToString(parts))
	}

	// Marshal new value
	valueBytes, err := yaml.Marshal(value)
	if err != nil {
		return err
	}

	var valueNode yaml.Node
	if err := yaml.Unmarshal(valueBytes, &valueNode); err != nil {
		return err
	}

	// Update the node in place, preserving style hints
	if valueNode.Kind == yaml.DocumentNode && len(valueNode.Content) > 0 {
		*node = *valueNode.Content[0]
	} else {
		node.Value = strings.TrimSpace(string(valueBytes))
	}

	return nil
}

func (w *LayerWriter) applyRemove(root *yaml.Node, parts []pathPart) error {
	if len(parts) == 0 {
		return fmt.Errorf("cannot remove root node")
	}

	// Find parent node
	parentParts := parts[:len(parts)-1]
	var parent *yaml.Node
	if len(parentParts) == 0 {
		if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
			parent = root.Content[0]
		} else {
			parent = root
		}
	} else {
		parent, _ = w.findNode(root, parentParts)
	}

	if parent == nil {
		return fmt.Errorf("parent path not found")
	}

	lastPart := parts[len(parts)-1]
	if lastPart.isIndex {
		// Remove from array
		if parent.Kind != yaml.SequenceNode {
			return fmt.Errorf("cannot remove index from non-sequence node")
		}
		if lastPart.index >= len(parent.Content) {
			return fmt.Errorf("index out of bounds")
		}
		parent.Content = append(parent.Content[:lastPart.index], parent.Content[lastPart.index+1:]...)
	} else {
		// Remove from map
		if parent.Kind != yaml.MappingNode {
			return fmt.Errorf("cannot remove key from non-mapping node")
		}
		for i := 0; i < len(parent.Content)-1; i += 2 {
			if parent.Content[i].Value == lastPart.key {
				parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
				return nil
			}
		}
		return fmt.Errorf("key not found: %s", lastPart.key)
	}

	return nil
}

func (w *LayerWriter) findNode(root *yaml.Node, parts []pathPart) (*yaml.Node, *yaml.Node) {
	if len(parts) == 0 {
		return root, nil
	}

	current := root
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	var parent *yaml.Node
	for i, part := range parts {
		parent = current

		if part.isIndex {
			if current.Kind != yaml.SequenceNode {
				return nil, parent
			}
			if part.index >= len(current.Content) {
				return nil, parent
			}
			current = current.Content[part.index]
		} else {
			if current.Kind != yaml.MappingNode {
				return nil, parent
			}
			found := false
			for j := 0; j < len(current.Content)-1; j += 2 {
				if current.Content[j].Value == part.key {
					current = current.Content[j+1]
					found = true
					break
				}
			}
			if !found {
				if i == len(parts)-1 {
					// Last part not found, return parent for add operation
					return nil, parent
				}
				return nil, parent
			}
		}
	}

	return current, parent
}

func pathToString(parts []pathPart) string {
	var result strings.Builder
	for i, part := range parts {
		if part.isIndex {
			fmt.Fprintf(&result, "[%d]", part.index)
		} else {
			if i > 0 {
				result.WriteString(".")
			}
			result.WriteString(part.key)
		}
	}
	return result.String()
}
