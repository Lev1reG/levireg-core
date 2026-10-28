package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"

	"github.com/Lev1reG/levireg-core/api"
)

type spec struct {
	Paths      map[string]map[string]any `yaml:"paths"`
	Components struct {
		Schemas struct {
			Code struct {
				Enum []string `yaml:"enum"`
			} `yaml:"Code"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

func loadSpec(t *testing.T) spec {
	t.Helper()
	var s spec
	if err := yaml.Unmarshal(api.Spec, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// Every router route must be documented and every documented operation must exist.
func TestSpecMatchesRoutes(t *testing.T) {
	s := loadSpec(t)
	want := map[string]bool{}
	for path, ops := range s.Paths {
		for m := range ops {
			want[strings.ToUpper(m)+" "+path] = true
		}
	}

	srv := New(zap.NewNop(), ":0", (*pgxpool.Pool)(nil))
	err := chi.Walk(srv.Handler.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if route == "/docs" || route == "/openapi.yaml" {
			return nil
		}
		key := method + " " + route
		if !want[key] {
			t.Errorf("route %s is missing from api/openapi.yaml", key)
		}
		delete(want, key)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for k := range want {
		t.Errorf("api/openapi.yaml documents %s but no such route exists", k)
	}
}

// The spec's Code enum must equal the Code* constants in internal/handler.
func TestSpecCodeEnumMatchesConstants(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "../handler/response.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Values) != 1 || !strings.HasPrefix(vs.Names[0].Name, "Code") {
			return true
		}
		if lit, ok := vs.Values[0].(*ast.BasicLit); ok {
			v, _ := strconv.Unquote(lit.Value)
			got = append(got, v)
		}
		return true
	})
	want := loadSpec(t).Components.Schemas.Code.Enum
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("spec Code enum = %v, handler constants = %v", want, got)
	}
}
