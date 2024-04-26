package collections

import (
	"fmt"
	"reflect"

	"github.com/goccy/go-json"
)

const (
	rootLevel = -1000
)

type TreeDataInterface[ID comparable] interface {
	Identifier() ID
	ParentIdentifier() *ID
}

func TreeOf[ID comparable, T TreeDataInterface[ID]](data ...T) (out *TreeNode[ID, T]) {
	out = &TreeNode[ID, T]{level: rootLevel}
	out.root = out
	if len(data) == 0 {
		return
	}

	var defaultIdVal ID
	out.index = make(map[ID]*TreeNode[ID, T], len(data))
	out.children = make(map[ID]*TreeNode[ID, T])

	for _, d := range data {
		id := d.Identifier()
		out.index[id] = &TreeNode[ID, T]{
			root: out, data: d, children: make(map[ID]*TreeNode[ID, T]),
		}
		if parentId := d.ParentIdentifier(); parentId == nil || *parentId == defaultIdVal {
			out.index[id].level = 1
			out.children[id] = out.index[id]
		}
	}

	for _, d := range data {
		var ok bool
		var n *TreeNode[ID, T]
		var id = d.Identifier()
		if parentId := d.ParentIdentifier(); parentId == nil || *parentId == defaultIdVal {
			continue
		} else if n, ok = out.index[id]; !ok {
			out.Add(d)
		}
		n.addToParent(d.ParentIdentifier())
		if n.ghost {
			n.unGhost(d)
		}
	}

	if len(out.children) == 1 {
		out = Map[ID, *TreeNode[ID, T]](out.children).Values().First()
		out.level = rootLevel
	}

	for _, r := range out.children {
		r.setLevel(1)
	}
	return
}

type TreeNode[ID comparable, T TreeDataInterface[ID]] struct {
	data     T
	level    int
	root     *TreeNode[ID, T]
	parent   *TreeNode[ID, T]
	children map[ID]*TreeNode[ID, T]

	// tracing
	ghost bool

	// root data
	index map[ID]*TreeNode[ID, T]
	// ghostParents map[ID]*TreeNode[ID, T]
}

func (t TreeNode[ID, T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.toData())
}

func (t *TreeNode[ID, T]) setParent(p *TreeNode[ID, T]) (affectedNodes int64) {
	t.parent = p
	t.setLevel(p.level + 1)
	return
}

func (t *TreeNode[ID, T]) setLevel(level int) (affectedNodes int64) {
	t.level = level
	if len(t.children) == 0 {
		return
	}
	// consideration: check if siblings are at the same level
	for _, c := range t.children {
		if c.level == level+1 {
			continue
		}
		affectedNodes += c.setLevel(level + 1)
	}
	return
}

func (t *TreeNode[ID, T]) isRoot() bool {
	return t.level == rootLevel
}

func (t TreeNode[ID, T]) Size() int {
	return len(t.Root().index)
}

func (t TreeNode[ID, T]) Value() T {
	return t.data
}

func (t TreeNode[ID, T]) Root() *TreeNode[ID, T] {
	return t.root
}

func (t TreeNode[ID, T]) Get(id ID) (out *TreeNode[ID, T]) {
	if len(t.children) == 0 {
		return
	}
	return t.children[id]
}

func (t TreeNode[ID, T]) Find(id ID) (out *TreeNode[ID, T]) {
	return t.Root().index[id]
}

func (t TreeNode[ID, T]) Distance(target *TreeNode[ID, T]) (dst int, ok bool) {
	if target == nil {
		return
	}
	return target.level - t.level, true
}

func (t TreeNode[ID, T]) DistanceOf(targetID ID) (dst int, ok bool) {
	var target *TreeNode[ID, T]
	if target, ok = t.index[targetID]; !ok || target == nil {
		return 0, false
	}
	return t.Distance(target)
}

func (t *TreeNode[ID, T]) Remove() (affectedNodes int64) {
	if len(t.children) > 0 {
		for _, c := range t.children {
			affectedNodes += c.Remove()
		}
	}
	if t.parent != nil {
		delete(t.parent.children, t.data.Identifier())
	}
	delete(t.Root().index, t.data.Identifier())
	return affectedNodes + 1
}

func (t *TreeNode[ID, T]) RemoveChild(id ID) (affectedNodes int64) {
	if len(t.children) == 0 {
		return
	} else if c, ok := t.children[id]; ok {
		return c.Remove()
	}
	return 
}

func (t *TreeNode[ID, T]) addToParent(parentId *ID) (affectedNodes int64) {
	var defaultIdVal ID
	if parentId == nil || *parentId == defaultIdVal {
		return
	}
	id := t.data.Identifier()
	if p, ok := t.Root().index[*parentId]; !ok {
		ghost := &TreeNode[ID, T]{
			root: t.Root(), children: map[ID]*TreeNode[ID, T]{id: t}, ghost: true,
		}
		t.Root().children[*parentId], t.Root().index[*parentId] = ghost, ghost
	} else {
		t.parent, p.children[id] = p, t
	}
	return 1
}

func (t *TreeNode[ID, T]) unGhost(data T) (affectedNodes int64) {
	id := data.Identifier()
	if _, ok := t.Root().index[id]; !ok {
		// what do we do ?
		t.Root().index[id] = &TreeNode[ID, T]{
			root: t.Root(), data: data, children: make(map[ID]*TreeNode[ID, T]), level: 1,
		}
	}
	t.data, t.ghost = data, false
	if affectedNodes += t.addToParent(data.ParentIdentifier()); affectedNodes > 0 {
		delete(t.Root().children, id)
		affectedNodes += t.setLevel(t.parent.level + 1)
	}
	return affectedNodes + 1
}

func (t *TreeNode[ID, T]) Add(d T) (affectedNodes int64) {
	var ok bool
	var affected int64
	var node *TreeNode[ID, T]
	if node, ok = t.Root().index[d.Identifier()]; ok {
		if parentId := d.ParentIdentifier(); parentId != nil && *parentId != *t.data.ParentIdentifier() {
			affectedNodes += node.addToParent(parentId)
		} else if _, isChild := t.children[d.Identifier()]; !isChild {
			t.children[d.Identifier()], node.parent = node, t
			affectedNodes += t.setLevel(t.parent.level + 1)
		}
		if !node.ghost {
			affectedNodes += node.unGhost(d)
		}
		return
	}
	node = &TreeNode[ID, T]{
		root: t.Root(), data: d, parent: t, level: t.level + 1,
	}
	t.Root().index[d.Identifier()] = node
	if t.isRoot() {
		affectedNodes += node.addToParent(d.ParentIdentifier())
	} else {
		t.children[d.Identifier()], node.parent = node, t
		affected ++
	}
	return affectedNodes + affected
}

func (t TreeNode[ID, T]) Json() (out []byte) {
	var err error
	if out, err = json.Marshal(t); err != nil {
		return []byte(fmt.Sprintf(`{"error":%s}`, err))
	}
	return
}

func (t TreeNode[ID, T]) toData() (out map[string]any) {
	out = make(map[string]interface{})
	var childLabel = "data"
	if t.level != rootLevel {
		out = t.asMap()
		childLabel = "children"
		if t.parent == nil || t.data.ParentIdentifier() == nil {
			ListOf("parentId", "parentID", "ParentID", "parent_id").
				ForEach(func(i int, field string) string { delete(out, field); return "" })
		}
	}

	if len(t.children) == 0 {
		return
	}
	children := make([]map[string]any, 0)
	for _, c := range t.children {
		children = append(children, c.toData())
	}
	if len(children) > 0 {
		out[childLabel] = children
	}
	return
}

func (tn TreeNode[ID, T]) asMap() (out map[string]any) {
	out = make(map[string]interface{})
	v := reflect.ValueOf(tn.data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fi := t.Field(i)
		if tagValue := fi.Tag.Get("json"); tagValue != "" {
			out[tagValue] = v.Field(i).Interface()
		}
	}
	return
}

func (t TreeNode[ID, T]) Clear() (affectedNodes int) {
	affectedNodes++
	if len(t.children) > 0 {
		for _, c := range t.children {
			affectedNodes += c.Clear()
		}
	}
	if t.parent != nil {
		delete(t.parent.children, t.data.Identifier())
	}
	if t.Root() != nil {
		delete(t.Root().index, t.data.Identifier())
	}
	return
}
