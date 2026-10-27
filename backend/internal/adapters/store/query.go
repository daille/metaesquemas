package store

import (
	"sort"
	"strings"

	"github.com/org/metaesquemas/internal/domain"
)

func (c *Catalog) Stats() domain.Stats {
	s := c.snap.Load()
	st := domain.Stats{ByTipo: map[string]int{"basico": 0, "estructurado": 0, "documental": 0, "servicio": 0}, Archivos: len(s.idx.Files), Actualizado: s.idx.GeneratedAt}
	resp := map[string]bool{}
	t := today()
	for _, e := range s.idx.Entries {
		st.Total++
		st.ByTipo[e.Tipo]++
		switch estadoVigencia(e.Vigencia, t) {
		case "vigente":
			st.Vigentes++
		case "vencida":
			st.Vencidos++
		}
		if e.Responsable != nil {
			resp[e.Responsable.Nombre] = true
		}
	}
	st.Responsables = len(resp)
	top := append([]domain.Entry(nil), s.idx.Entries...)
	sort.SliceStable(top, func(i, j int) bool { return top[i].UsedByCount > top[j].UsedByCount })
	for _, e := range top {
		if e.UsedByCount == 0 || len(st.MasUsados) == 5 {
			break
		}
		st.MasUsados = append(st.MasUsados, c.withState(e))
	}
	if st.MasUsados == nil {
		st.MasUsados = []domain.Entry{}
	}
	return st
}

func (c *Catalog) Facets() domain.Facets {
	s := c.snap.Load()
	tipos, resp, vig, usada := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	t := today()
	for _, e := range s.idx.Entries {
		tipos[e.Tipo]++
		if e.Responsable != nil {
			resp[e.Responsable.Nombre]++
		}
		vig[estadoVigencia(e.Vigencia, t)]++
		if e.UsedByCount > 0 {
			usada["si"]++
		} else {
			usada["no"]++
		}
	}
	return domain.Facets{Tipos: sortedKeys(tipos), Responsables: sortedKeys(resp), Vigencia: sortedKeys(vig), Usada: sortedKeys(usada)}
}

// Namespaces arma el árbol de carpetas/namespaces con la cantidad de esquemas.
func (c *Catalog) Namespaces() []*domain.NsNode {
	s := c.snap.Load()
	root := &domain.NsNode{}
	for _, e := range s.idx.Entries {
		cur := root
		path := ""
		for _, seg := range strings.Split(e.Namespace, ".") {
			if path != "" {
				path += "."
			}
			path += seg
			var next *domain.NsNode
			for _, ch := range cur.Children {
				if ch.Name == seg {
					next = ch
					break
				}
			}
			if next == nil {
				next = &domain.NsNode{Name: seg, Path: path}
				cur.Children = append(cur.Children, next)
			}
			next.Count++
			cur = next
		}
	}
	var sortRec func(n *domain.NsNode)
	sortRec = func(n *domain.NsNode) {
		sort.Slice(n.Children, func(i, j int) bool { return n.Children[i].Name < n.Children[j].Name })
		for _, ch := range n.Children {
			sortRec(ch)
		}
	}
	sortRec(root)
	if root.Children == nil {
		return []*domain.NsNode{}
	}
	return root.Children
}

func (c *Catalog) List(p domain.ListParams) domain.ListResult {
	s := c.snap.Load()
	t := today()
	tokens := strings.Fields(fold(p.Q))
	tipos := map[string]bool{}
	for _, x := range p.Tipos {
		tipos[x] = true
	}
	var out []domain.Entry
	for _, e := range s.idx.Entries {
		if len(tipos) > 0 && !tipos[e.Tipo] {
			continue
		}
		if p.Responsable != "" && (e.Responsable == nil || e.Responsable.Nombre != p.Responsable) {
			continue
		}
		if p.Namespace != "" && e.Namespace != p.Namespace && !strings.HasPrefix(e.Namespace, p.Namespace+".") {
			continue
		}
		if p.Vigencia != "" && estadoVigencia(e.Vigencia, t) != p.Vigencia {
			continue
		}
		if (p.Usada == "si" && e.UsedByCount == 0) || (p.Usada == "no" && e.UsedByCount > 0) {
			continue
		}
		hay := s.hay[e.ID]
		ok := true
		for _, tok := range tokens {
			if !strings.Contains(hay, tok) {
				ok = false
				break
			}
		}
		if ok {
			e.VigenciaEstado = estadoVigencia(e.Vigencia, t)
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		switch p.Sort {
		case "usos":
			if out[i].UsedByCount != out[j].UsedByCount {
				return out[i].UsedByCount > out[j].UsedByCount
			}
		case "responsable":
			if a, b := respName(out[i]), respName(out[j]); a != b {
				return a < b
			}
		case "namespace":
			if out[i].Namespace != out[j].Namespace {
				return out[i].Namespace < out[j].Namespace
			}
		}
		return out[i].ID < out[j].ID
	})
	total := len(out)
	lo, hi := p.Offset, p.Offset+p.Limit
	if lo > total {
		lo = total
	}
	if hi > total {
		hi = total
	}
	items := out[lo:hi]
	if items == nil {
		items = []domain.Entry{}
	}
	return domain.ListResult{Items: items, Total: total, Limit: p.Limit, Offset: p.Offset}
}

func respName(e domain.Entry) string {
	if e.Responsable == nil {
		return ""
	}
	return e.Responsable.Nombre
}
