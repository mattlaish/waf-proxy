package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

type contractSchemaNode struct {
	Type     string
	Format   string
	Required bool
	Enum     []string
	Array    bool
	Children map[string]*contractSchemaNode
}

func (s *contractStore) exportDocument(contractID, versionID, format string) ([]byte, string, error) {
	if s == nil {
		return nil, "", errors.New("contract store unavailable")
	}
	contract, ok := s.get(contractID)
	if !ok {
		return nil, "", errors.New("contract not found")
	}
	var version APIContractVersion
	var ops []ContractOperation
	if versionID == "" {
		var found bool
		version, ops, found = s.latestVersion(contractID)
		if !found {
			return nil, "", errors.New("contract has no versions")
		}
	} else {
		var found bool
		version, ops, found = s.version(versionID)
		if !found || version.ContractID != contractID {
			return nil, "", errors.New("contract version not found")
		}
	}

	root := map[string]any{
		"openapi":      version.OpenAPIVersion,
		"info":         map[string]any{"title": contract.Name, "version": version.Version},
		"paths":        map[string]any{},
		"x-waf-export": map[string]any{"contract_id": contract.ID, "version_id": version.ID, "content_hash": version.ContentHash},
	}
	if len(version.SecuritySchemes) > 0 {
		schemes := map[string]any{}
		keys := make([]string, 0, len(version.SecuritySchemes))
		for name := range version.SecuritySchemes {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			schemes[name] = cloneAnyMap(version.SecuritySchemes[name])
		}
		root["components"] = map[string]any{"securitySchemes": schemes}
	}
	if contract.Description != "" {
		root["info"].(map[string]any)["description"] = contract.Description
	}
	paths := root["paths"].(map[string]any)
	for _, op := range ops {
		pathItem, _ := paths[op.Path].(map[string]any)
		if pathItem == nil {
			pathItem = map[string]any{}
			paths[op.Path] = pathItem
		}
		o := map[string]any{"operationId": op.OperationID, "responses": map[string]any{}}
		if op.Summary != "" {
			o["summary"] = op.Summary
		}
		if len(op.Parameters) > 0 {
			o["parameters"] = exportParameters(op.Parameters)
		}
		if len(op.Security) > 0 {
			o["security"] = exportSecurity(op.Security)
		}
		if len(op.RequestFields) > 0 {
			mediaTypes := append([]string(nil), op.RequestMediaTypes...)
			if len(mediaTypes) == 0 && op.RequestMedia != "" {
				mediaTypes = []string{op.RequestMedia}
			}
			if len(mediaTypes) == 0 {
				mediaTypes = []string{"application/json"}
			}
			content := map[string]any{}
			for _, media := range mediaTypes {
				content[media] = map[string]any{"schema": schemaFromFlatFields(op.RequestFields)}
			}
			o["requestBody"] = map[string]any{"content": content}
		}
		responses := o["responses"].(map[string]any)
		for status, fields := range op.ResponseFields {
			mediaTypes := append([]string(nil), op.ResponseMediaTypes[status]...)
			if len(mediaTypes) == 0 && op.ResponseMedia[status] != "" {
				mediaTypes = []string{op.ResponseMedia[status]}
			}
			if len(mediaTypes) == 0 {
				mediaTypes = []string{"application/json"}
			}
			resp := map[string]any{"description": "Observed contract response"}
			if len(fields) > 0 {
				content := map[string]any{}
				for _, media := range mediaTypes {
					content[media] = map[string]any{"schema": schemaFromFlatFields(fields)}
				}
				resp["content"] = content
			}
			responses[status] = resp
		}
		if len(responses) == 0 {
			responses["default"] = map[string]any{"description": "Response"}
		}
		pathItem[strings.ToLower(op.Method)] = o
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "yaml" || format == "yml" {
		b, err := yaml.Marshal(root)
		return b, "application/yaml; charset=utf-8", err
	}
	b, err := json.MarshalIndent(root, "", "  ")
	return b, "application/json; charset=utf-8", err
}

func exportParameters(params []ContractParameter) []any {
	out := make([]any, 0, len(params))
	for _, p := range params {
		schema := map[string]any{}
		if p.Type != "" {
			schema["type"] = p.Type
		}
		if p.Format != "" {
			schema["format"] = p.Format
		}
		if len(p.Enum) > 0 {
			schema["enum"] = p.Enum
		}
		out = append(out, map[string]any{"name": p.Name, "in": p.In, "required": p.Required, "schema": schema})
	}
	return out
}

func exportSecurity(sec []ContractSecurityRequirement) []any {
	grouped := map[int]map[string][]string{}
	for _, r := range sec {
		if grouped[r.Group] == nil {
			grouped[r.Group] = map[string][]string{}
		}
		if r.Scheme != "" {
			grouped[r.Group][r.Scheme] = append([]string(nil), r.Scopes...)
		}
	}
	groups := make([]int, 0, len(grouped))
	for group := range grouped {
		groups = append(groups, group)
	}
	sort.Ints(groups)
	out := make([]any, 0, len(groups))
	for _, group := range groups {
		m := map[string]any{}
		keys := make([]string, 0, len(grouped[group]))
		for k := range grouped[group] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			m[k] = grouped[group][k]
		}
		out = append(out, m)
	}
	return out
}

func schemaFromFlatFields(fields []ContractSchemaField) map[string]any {
	root := &contractSchemaNode{Type: "object", Children: map[string]*contractSchemaNode{}}
	for _, f := range fields {
		insertFlatContractField(root, f)
	}
	return contractSchemaNodeMap(root)
}

func insertFlatContractField(root *contractSchemaNode, f ContractSchemaField) {
	path := strings.TrimSpace(f.Path)
	if path == "" || path == "$" {
		root.Type = f.Type
		root.Format = f.Format
		root.Enum = append([]string(nil), f.Enum...)
		root.Required = f.Required
		return
	}
	parts := strings.Split(path, ".")
	cur := root
	for i, raw := range parts {
		arr := strings.HasSuffix(raw, "[]")
		name := strings.TrimSuffix(raw, "[]")
		if name == "" {
			name = "items"
		}
		if cur.Children == nil {
			cur.Children = map[string]*contractSchemaNode{}
		}
		n := cur.Children[name]
		if n == nil {
			n = &contractSchemaNode{Children: map[string]*contractSchemaNode{}}
			cur.Children[name] = n
		}
		if arr {
			n.Array = true
		}
		if i == len(parts)-1 {
			n.Type = f.Type
			n.Format = f.Format
			n.Required = f.Required
			n.Enum = append([]string(nil), f.Enum...)
		}
		cur = n
	}
}

func contractSchemaNodeMap(n *contractSchemaNode) map[string]any {
	if n == nil {
		return map[string]any{}
	}
	if n.Array {
		clone := *n
		clone.Array = false
		return map[string]any{"type": "array", "items": contractSchemaNodeMap(&clone)}
	}
	m := map[string]any{}
	if len(n.Children) > 0 {
		m["type"] = "object"
		props := map[string]any{}
		required := []string{}
		keys := make([]string, 0, len(n.Children))
		for k := range n.Children {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := n.Children[k]
			props[k] = contractSchemaNodeMap(child)
			if child.Required {
				required = append(required, k)
			}
		}
		m["properties"] = props
		if len(required) > 0 {
			m["required"] = required
		}
	} else if n.Type != "" {
		m["type"] = n.Type
	} else {
		m["type"] = "string"
	}
	if n.Format != "" {
		m["format"] = n.Format
	}
	if len(n.Enum) > 0 {
		m["enum"] = n.Enum
	}
	return m
}

func (a *adminServer) handleContractExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	b, contentType, err := a.srv.contracts.exportDocument(id, r.URL.Query().Get("version_id"), r.URL.Query().Get("format"))
	if err != nil {
		status := httpStatusForContractExportError(err)
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="openapi-%s.%s"`, id, exportExtension(contentType)))
	_, _ = w.Write(b)
}

func httpStatusForContractExportError(err error) int {
	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no versions") {
		return 404
	}
	return 500
}
func exportExtension(contentType string) string {
	if strings.Contains(contentType, "yaml") {
		return "yaml"
	}
	return "json"
}
