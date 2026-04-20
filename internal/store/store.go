package store

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
)

type Filter struct {
	Kind      *model.Kind
	Status    *model.Status
	Tag       string
	ProjectID string
}

// ItemWithBody bundles an item with its Markdown body as read from disk.
// Callers that need body-level predicates (e.g. the --query DSL) should
// use ListWithBody so the body is returned alongside the item in a single
// read per file.
type ItemWithBody struct {
	Item *model.Item
	Body string
}

func Read(path string) (*model.Item, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return parse(data)
}

func Write(path string, item *model.Item, body string) error {
	data, err := marshal(item, body)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Move(src, dst string, item *model.Item, body string) error {
	if err := Write(dst, item, body); err != nil {
		return err
	}
	return os.Remove(src)
}

func List(cfg *config.Config, filter Filter) ([]*model.Item, error) {
	results, err := listScan(cfg, filter)
	if err != nil {
		return nil, err
	}
	items := make([]*model.Item, len(results))
	for i, r := range results {
		items[i] = r.Item
	}
	return items, nil
}

// ListWithBody is like List but returns each item alongside its Markdown
// body, read in the same pass. Results are sorted by CreatedAt ascending.
func ListWithBody(cfg *config.Config, filter Filter) ([]ItemWithBody, error) {
	return listScan(cfg, filter)
}

func listScan(cfg *config.Config, filter Filter) ([]ItemWithBody, error) {
	var dirs []string

	if filter.Kind != nil {
		dirs = []string{cfg.DirForKind(*filter.Kind)}
	} else {
		for _, k := range model.ValidKinds() {
			dirs = append(dirs, cfg.DirForKind(k))
		}
	}

	// Include archive when status filter allows non-active or no status filter
	includeArchive := filter.Status == nil || model.IsTerminal(*filter.Status)
	if includeArchive {
		dirs = append(dirs, cfg.ArchiveItemsDir())
	}

	var results []ItemWithBody
	seen := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if seen[path] {
				continue
			}
			seen[path] = true

			item, body, err := Read(path)
			if err != nil {
				return nil, err
			}

			if !matchFilter(item, filter) {
				continue
			}
			results = append(results, ItemWithBody{Item: item, Body: body})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Item.CreatedAt.Before(results[j].Item.CreatedAt)
	})
	return results, nil
}

func matchFilter(item *model.Item, f Filter) bool {
	if f.Kind != nil && item.Kind != *f.Kind {
		return false
	}
	if f.Status != nil && item.Status != *f.Status {
		return false
	}
	if f.Tag != "" && !hasTag(item, f.Tag) {
		return false
	}
	if f.ProjectID != "" && item.Project != f.ProjectID {
		return false
	}
	return true
}

func hasTag(item *model.Item, tag string) bool {
	return slices.Contains(item.Tags, tag)
}

// parse splits YAML front matter from body.
// Front matter is delimited by lines that are exactly "---".
func parse(data []byte) (*model.Item, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	var yamlLines []string
	var bodyLines []string

	state := 0 // 0=before first ---, 1=in front matter, 2=in body
	for scanner.Scan() {
		line := scanner.Text()
		switch state {
		case 0:
			if line == "---" {
				state = 1
			}
		case 1:
			if line == "---" {
				state = 2
			} else {
				yamlLines = append(yamlLines, line)
			}
		case 2:
			bodyLines = append(bodyLines, line)
		}
	}

	var item model.Item
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &item); err != nil {
		return nil, "", err
	}

	body := strings.Join(bodyLines, "\n")
	// Trim leading newline that follows the closing ---
	body = strings.TrimPrefix(body, "\n")

	return &item, body, nil
}

func marshal(item *model.Item, body string) ([]byte, error) {
	yamlData, err := yaml.Marshal(item)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(yamlData)
	sb.WriteString("---\n")
	if body != "" {
		sb.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			sb.WriteString("\n")
		}
	}
	return []byte(sb.String()), nil
}
