package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
)

// ReferenceWithBody bundles a Reference with its Markdown body as read from
// disk. Returned by listing and rendering helpers that need to inspect body
// content (e.g. INDEX.md description extraction).
type ReferenceWithBody struct {
	Reference *model.Reference
	Body      string
	Tool      string
	Archived  bool
}

// ReadRef reads a reference Markdown file at path.
func ReadRef(path string) (*model.Reference, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	yamlBytes, body := splitFrontmatter(data)
	var ref model.Reference
	if err := yaml.Unmarshal(yamlBytes, &ref); err != nil {
		return nil, "", err
	}
	return &ref, body, nil
}

// WriteRef writes a reference to path atomically.
func WriteRef(path string, ref *model.Reference, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := marshalFrontmatter(ref, body)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// MoveRef writes the reference at dst and removes the file at src. Used for
// archive/restore.
func MoveRef(src, dst string, ref *model.Reference, body string) error {
	if err := WriteRef(dst, ref, body); err != nil {
		return err
	}
	return os.Remove(src)
}

// ArchiveReference moves an active reference to archive/reference/<tool>/.
// References have no `status` field; archival is location-only. The caller is
// responsible for rewriting the active INDEX.md afterwards.
func ArchiveReference(cfg *config.Config, tool string, ref *model.Reference, body string) error {
	src := PathForReferenceActive(cfg, tool, ref.ID)
	dst := PathForReferenceArchive(cfg, tool, ref.ID)
	return MoveRef(src, dst, ref, body)
}

// RestoreReference moves an archived reference back to reference/<tool>/.
// Symmetric inverse of ArchiveReference. The caller is responsible for
// rewriting the active INDEX.md afterwards.
func RestoreReference(cfg *config.Config, tool string, ref *model.Reference, body string) error {
	src := PathForReferenceArchive(cfg, tool, ref.ID)
	dst := PathForReferenceActive(cfg, tool, ref.ID)
	return MoveRef(src, dst, ref, body)
}

// PathForReferenceActive returns the canonical active path for a reference
// owned by tool.
func PathForReferenceActive(cfg *config.Config, tool, id string) string {
	return filepath.Join(cfg.ReferenceToolDir(tool), id+".md")
}

// PathForReferenceArchive returns the canonical archive path for a reference
// owned by tool.
func PathForReferenceArchive(cfg *config.Config, tool, id string) string {
	return filepath.Join(cfg.ArchiveReferenceToolDir(tool), id+".md")
}

// FindReferenceResult is the lookup result returned by FindReference.
type FindReferenceResult struct {
	Path     string
	Tool     string
	Archived bool
}

// FindReference locates a reference by ID across every tool directory under
// reference/<tool>/ and archive/reference/<tool>/. Active hits are preferred
// over archived hits. Returns NotFoundError when the ID cannot be found.
func FindReference(cfg *config.Config, id string) (FindReferenceResult, error) {
	filename := id + ".md"

	// Active first.
	tools, err := listToolDirs(cfg.ReferenceDir())
	if err != nil {
		return FindReferenceResult{}, err
	}
	for _, tool := range tools {
		p := filepath.Join(cfg.ReferenceToolDir(tool), filename)
		if _, err := os.Stat(p); err == nil {
			return FindReferenceResult{Path: p, Tool: tool, Archived: false}, nil
		}
	}

	// Archive fallback.
	archTools, err := listToolDirs(cfg.ArchiveReferenceDir())
	if err != nil {
		return FindReferenceResult{}, err
	}
	for _, tool := range archTools {
		p := filepath.Join(cfg.ArchiveReferenceToolDir(tool), filename)
		if _, err := os.Stat(p); err == nil {
			return FindReferenceResult{Path: p, Tool: tool, Archived: true}, nil
		}
	}

	return FindReferenceResult{}, &NotFoundError{Kind: EntityReference, ID: id}
}

// ReferenceExists reports whether any reference (active or archived, in any
// tool) has the given ID.
func ReferenceExists(cfg *config.Config, id string) bool {
	_, err := FindReference(cfg, id)
	return err == nil
}

// ListReferences returns active references for the given tool sorted by
// (UpdatedAt desc, ID asc). When includeArchive is true, archived references
// for the same tool are appended (still sorted within the active and archive
// halves the same way).
func ListReferences(cfg *config.Config, tool string, includeArchive bool) ([]ReferenceWithBody, error) {
	active, err := readToolDir(cfg.ReferenceToolDir(tool), tool, false)
	if err != nil {
		return nil, err
	}
	sortReferences(active)
	if !includeArchive {
		return active, nil
	}
	archived, err := readToolDir(cfg.ArchiveReferenceToolDir(tool), tool, true)
	if err != nil {
		return nil, err
	}
	sortReferences(archived)
	return append(active, archived...), nil
}

// ListReferenceTools returns the set of tools that have any active reference
// directory under reference/. Useful for cross-tool ID lookups.
func ListReferenceTools(cfg *config.Config) ([]string, error) {
	return listToolDirs(cfg.ReferenceDir())
}

// EnsureReferenceToolDir creates reference/<tool>/ if it does not yet exist.
// Lazy creation chosen over a strict precondition so users do not have to run
// a setup step before their first `htd reference add --tool foo`.
func EnsureReferenceToolDir(cfg *config.Config, tool string) error {
	return os.MkdirAll(cfg.ReferenceToolDir(tool), 0o755)
}

// listToolDirs returns the names of every immediate subdirectory of root.
// Used to discover tools without prescribing them.
func listToolDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var tools []string
	for _, e := range entries {
		if e.IsDir() {
			tools = append(tools, e.Name())
		}
	}
	sort.Strings(tools)
	return tools, nil
}

func readToolDir(dir, tool string, archived bool) ([]ReferenceWithBody, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ReferenceWithBody
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// Skip generated INDEX.md.
		if e.Name() == "INDEX.md" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		ref, body, err := ReadRef(path)
		if err != nil {
			return nil, err
		}
		out = append(out, ReferenceWithBody{
			Reference: ref,
			Body:      body,
			Tool:      tool,
			Archived:  archived,
		})
	}
	return out, nil
}

func sortReferences(refs []ReferenceWithBody) {
	sort.SliceStable(refs, func(i, j int) bool {
		ti := refs[i].Reference.UpdatedAt
		tj := refs[j].Reference.UpdatedAt
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return refs[i].Reference.ID < refs[j].Reference.ID
	})
}
