// Package application valida los parámetros y orquesta los puertos. No compila ni transforma esquemas:
// la compilación ocurre en el worker al cargar el archivo.
package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/org/metaesquemas/internal/domain"
)

var (
	idRe       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
	fileNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,80}\.tsp$`)
	nsRe       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
	uploaderRe = regexp.MustCompile(`[^\p{L}\p{N} ._@-]`)
)

var (
	tipos      = map[string]bool{"basico": true, "estructurado": true, "documental": true, "servicio": true}
	vigencias  = map[string]bool{"vigente": true, "vencida": true, "futura": true, "sin-vigencia": true}
	sorts      = map[string]bool{"": true, "nombre": true, "usos": true, "responsable": true, "namespace": true}
	artifacts  = map[string]bool{"tsp": true, "tsp-file": true, "jsonschema": true, "xsd": true, "proto": true, "openapi": true}
	directions = map[string]bool{"uses": true, "usedBy": true}
)

type Service struct {
	Catalog   domain.Catalog
	Ingest    domain.Ingest
	MaxUpload int64
}

func bad(f string, a ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalid, fmt.Sprintf(f, a...))
}

func (s *Service) checkID(id string) error {
	if len(id) > 300 || !idRe.MatchString(id) {
		return bad("identificador de esquema inválido")
	}
	return nil
}

func (s *Service) List(p domain.ListParams) (domain.ListResult, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	if p.Offset < 0 || p.Offset > 100000 {
		return domain.ListResult{}, bad("offset fuera de rango")
	}
	if len(p.Q) > 200 {
		return domain.ListResult{}, bad("texto de búsqueda demasiado largo")
	}
	for _, t := range p.Tipos {
		if !tipos[t] {
			return domain.ListResult{}, bad("tipo inválido: %q", t)
		}
	}
	if p.Vigencia != "" && !vigencias[p.Vigencia] {
		return domain.ListResult{}, bad("vigencia inválida (vigente, vencida, futura, sin-vigencia)")
	}
	if p.Usada != "" && p.Usada != "si" && p.Usada != "no" {
		return domain.ListResult{}, bad("usada debe ser si o no")
	}
	if !sorts[p.Sort] {
		return domain.ListResult{}, bad("orden inválido")
	}
	if p.Namespace != "" && (len(p.Namespace) > 200 || !nsRe.MatchString(p.Namespace)) {
		return domain.ListResult{}, bad("namespace inválido")
	}
	return s.Catalog.List(p), nil
}

func (s *Service) Get(id string, version int) (*domain.Detail, error) {
	if err := s.checkID(id); err != nil {
		return nil, err
	}
	if version < 0 || version > 100000 {
		return nil, bad("versión inválida")
	}
	return s.Catalog.Get(id, version)
}

func (s *Service) Artifact(id, kind string, version int) (domain.Artifact, error) {
	if err := s.checkID(id); err != nil {
		return domain.Artifact{}, err
	}
	if !artifacts[kind] {
		return domain.Artifact{}, bad("proyección inválida (tsp, tsp-file, jsonschema, xsd, proto, openapi)")
	}
	if version < 0 || version > 100000 {
		return domain.Artifact{}, bad("versión inválida")
	}
	return s.Catalog.Artifact(id, kind, version)
}

func (s *Service) Tree(id, direction string, depth int) (*domain.TreeNode, error) {
	if err := s.checkID(id); err != nil {
		return nil, err
	}
	if direction == "" {
		direction = "uses"
	}
	if !directions[direction] {
		return nil, bad("direction debe ser uses o usedBy")
	}
	return s.Catalog.Tree(id, direction, depth)
}

func (s *Service) Graph(id string, depth int) (*domain.Graph, error) {
	if err := s.checkID(id); err != nil {
		return nil, err
	}
	return s.Catalog.Graph(id, depth)
}

// Upload encola un archivo .tsp para que el worker lo valide, compile y publique.
func (s *Service) Upload(ctx context.Context, fileName, uploader string, source []byte) (string, error) {
	if !fileNameRe.MatchString(fileName) {
		return "", bad("nombre de archivo inválido: use letras, números, punto o guion y la extensión .tsp")
	}
	if len(source) == 0 || int64(len(source)) > s.MaxUpload {
		return "", bad("el archivo debe pesar entre 1 byte y %d bytes", s.MaxUpload)
	}
	if !utf8.Valid(source) || strings.ContainsRune(string(source), 0) {
		return "", bad("el archivo debe ser texto UTF-8")
	}
	uploader = uploaderRe.ReplaceAllString(uploader, "")
	if len(uploader) > 64 {
		uploader = uploader[:64]
	}
	return s.Ingest.Submit(ctx, fileName, uploader, source)
}
