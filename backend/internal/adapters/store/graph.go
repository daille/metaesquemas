package store

import (
	"sort"

	"github.com/org/metaesquemas/internal/domain"
)

const (
	maxDepth = 6
	maxNodes = 400
)

// links agrupa las relaciones con un mismo esquema; `relation` es la principal (la que define el estilo).
type links struct {
	id, relation string
	relations    []string
}

var priority = map[string]int{"extension": 5, "composicion": 4, "variante": 3, "retorno": 2, "uso": 1}

func merge(raw []links) []links {
	byID := map[string]*links{}
	var order []string
	for _, l := range raw {
		cur, ok := byID[l.id]
		if !ok {
			cur = &links{id: l.id, relation: l.relation}
			byID[l.id] = cur
			order = append(order, l.id)
		}
		dup := false
		for _, r := range cur.relations {
			dup = dup || r == l.relation
		}
		if !dup {
			cur.relations = append(cur.relations, l.relation)
		}
		if priority[l.relation] > priority[cur.relation] {
			cur.relation = l.relation
		}
	}
	out := make([]links, 0, len(order))
	for _, id := range order {
		sort.Strings(byID[id].relations)
		out = append(out, *byID[id])
	}
	return out
}

func (s *snapshot) neighbors(id, direction string) []links {
	e := s.byID[id]
	if e == nil {
		return nil
	}
	var out []links
	if direction == "usedBy" {
		for _, u := range e.UsedBy {
			out = append(out, links{id: u.ID, relation: u.Relation})
		}
	} else {
		for _, d := range e.Dependencies {
			out = append(out, links{id: d.ID, relation: d.Relation})
		}
	}
	return merge(out)
}

// Tree devuelve el árbol de dependencias (direction="uses") o de uso inverso (direction="usedBy"),
// marcando ciclos y acotando profundidad y tamaño.
func (c *Catalog) Tree(id, direction string, depth int) (*domain.TreeNode, error) {
	s := c.snap.Load()
	root := s.byID[id]
	if root == nil {
		return nil, domain.ErrNotFound
	}
	if depth < 1 || depth > maxDepth {
		depth = 3
	}
	count := 0
	var build func(id, relation string, relations []string, d int, path map[string]bool) *domain.TreeNode
	build = func(id, relation string, relations []string, d int, path map[string]bool) *domain.TreeNode {
		count++
		e := s.byID[id]
		n := &domain.TreeNode{ID: id, Relation: relation, Relations: relations, Exists: e != nil, Name: lastSegment(id)}
		if e == nil {
			return n // referencia a un esquema que aún no está en el catálogo
		}
		n.Name, n.Tipo = e.Name, e.Tipo
		if path[id] {
			n.Cycle = true
			return n
		}
		nb := s.neighbors(id, direction)
		if d == 0 {
			n.Truncated = len(nb) > 0
			return n
		}
		path[id] = true
		sort.Slice(nb, func(i, j int) bool { return nb[i].id < nb[j].id })
		for _, l := range nb {
			if count >= maxNodes {
				n.Truncated = true
				break
			}
			n.Children = append(n.Children, build(l.id, l.relation, l.relations, d-1, path))
		}
		delete(path, id)
		return n
	}
	return build(id, "", nil, depth, map[string]bool{}), nil
}

func lastSegment(id string) string {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '.' {
			return id[i+1:]
		}
	}
	return id
}

// Graph devuelve el vecindario (uso y dependencias) hasta `depth` saltos, para el grafo de composición.
func (c *Catalog) Graph(id string, depth int) (*domain.Graph, error) {
	s := c.snap.Load()
	if s.byID[id] == nil {
		return nil, domain.ErrNotFound
	}
	if depth < 1 || depth > 3 {
		depth = 1
	}
	g := &domain.Graph{Center: id, Nodes: []domain.GraphNode{}, Edges: []domain.GraphEdge{}}
	seenN := map[string]bool{}
	seenE := map[string]bool{}
	addNode := func(nid string) {
		if seenN[nid] {
			return
		}
		seenN[nid] = true
		if e := s.byID[nid]; e != nil {
			g.Nodes = append(g.Nodes, domain.GraphNode{ID: nid, Name: e.Name, Tipo: e.Tipo, Namespace: e.Namespace, Exists: true})
		} else {
			g.Nodes = append(g.Nodes, domain.GraphNode{ID: nid, Name: lastSegment(nid), Exists: false})
		}
	}
	addEdge := func(from, to string, l links) {
		k := from + "|" + to
		if !seenE[k] {
			seenE[k] = true
			g.Edges = append(g.Edges, domain.GraphEdge{Source: from, Target: to, Relation: l.relation, Relations: l.relations})
		}
	}
	frontier := []string{id}
	addNode(id)
	for d := 0; d < depth && len(g.Nodes) < maxNodes; d++ {
		var next []string
		for _, cur := range frontier {
			for _, l := range s.neighbors(cur, "uses") { // cur -> l.id
				if len(g.Nodes) >= maxNodes {
					break
				}
				if !seenN[l.id] {
					next = append(next, l.id)
				}
				addNode(l.id)
				addEdge(cur, l.id, l)
			}
			for _, l := range s.neighbors(cur, "usedBy") { // l.id -> cur
				if len(g.Nodes) >= maxNodes {
					break
				}
				if !seenN[l.id] {
					next = append(next, l.id)
				}
				addNode(l.id)
				addEdge(l.id, cur, l)
			}
		}
		frontier = next
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	return g, nil
}
