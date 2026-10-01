// development: lists the loops over Go maps in the game's code, by syntax only (names declared as
// maps, functions that return maps), and tags the ones whose body draws chance (DRAWS), stops at a
// match (STOPS) or builds a list (APPENDS). Go ranges over a map in an order of its own chance, so
// such a loop can make a run of one seed play two games (docs/mod/testing.md). Names are matched
// without types, so a name that is a map somewhere is flagged everywhere: read every hit.
//
//     go run ./util/mapranges game/magic | grep DRAWS
package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

func isMap(expr ast.Expr) bool {
    switch t := expr.(type) {
        case *ast.MapType: return true
        case *ast.StarExpr: return isMap(t.X)
    }
    return false
}

func main() {
    root := os.Args[1]
    fset := token.NewFileSet()
    var files []*ast.File
    var names []string
    filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err == nil && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") && !strings.Contains(path, "util") {
            f, e := parser.ParseFile(fset, path, nil, 0)
            if e == nil { files = append(files, f); names = append(names, path) }
        }
        return nil
    })
    mapNames := map[string]bool{}   // fields, vars and params declared as maps
    mapFuncs := map[string]bool{}   // functions and methods returning a map first
    mapTypes := map[string]bool{}   // named types that are maps
    for _, f := range files {
        ast.Inspect(f, func(n ast.Node) bool {
            switch d := n.(type) {
                case *ast.TypeSpec: if isMap(d.Type) { mapTypes[d.Name.Name] = true }
            }
            return true
        })
    }
    isMapT := func(e ast.Expr) bool {
        if isMap(e) { return true }
        if id, ok := e.(*ast.Ident); ok { return mapTypes[id.Name] }
        if st, ok := e.(*ast.StarExpr); ok { if id, ok := st.X.(*ast.Ident); ok { return mapTypes[id.Name] } }
        return false
    }
    for _, f := range files {
        ast.Inspect(f, func(n ast.Node) bool {
            switch d := n.(type) {
                case *ast.Field: if isMapT(d.Type) { for _, nm := range d.Names { mapNames[nm.Name] = true } }
                case *ast.ValueSpec: if d.Type != nil && isMapT(d.Type) { for _, nm := range d.Names { mapNames[nm.Name] = true } }
                    for i, v := range d.Values { if c, ok := v.(*ast.CallExpr); ok { if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "make" && len(c.Args) > 0 && isMapT(c.Args[0]) && i < len(d.Names) { mapNames[d.Names[i].Name] = true } }; if cl, ok := v.(*ast.CompositeLit); ok && cl.Type != nil && isMapT(cl.Type) && i < len(d.Names) { mapNames[d.Names[i].Name] = true } }
                case *ast.AssignStmt:
                    for i, v := range d.Rhs {
                        if i >= len(d.Lhs) { break }
                        id, ok := d.Lhs[i].(*ast.Ident); if !ok { continue }
                        if c, ok := v.(*ast.CallExpr); ok { if fid, ok := c.Fun.(*ast.Ident); ok && fid.Name == "make" && len(c.Args) > 0 && isMapT(c.Args[0]) { mapNames[id.Name] = true } }
                        if cl, ok := v.(*ast.CompositeLit); ok && cl.Type != nil && isMapT(cl.Type) { mapNames[id.Name] = true }
                    }
                case *ast.FuncDecl:
                    if d.Type.Results != nil && len(d.Type.Results.List) > 0 && isMapT(d.Type.Results.List[0].Type) { mapFuncs[d.Name.Name] = true }
            }
            return true
        })
    }
    var out []string
    for i, f := range files {
        rel, _ := filepath.Rel(root, names[i])
        ast.Inspect(f, func(n ast.Node) bool {
            r, ok := n.(*ast.RangeStmt)
            if !ok { return true }
            hit := ""
            switch x := r.X.(type) {
                case *ast.Ident: if mapNames[x.Name] { hit = x.Name }
                case *ast.SelectorExpr: if mapNames[x.Sel.Name] { hit = x.Sel.Name }
                case *ast.CompositeLit: if x.Type != nil && isMapT(x.Type) { hit = "map literal" }
                case *ast.CallExpr:
                    switch fn := x.Fun.(type) {
                        case *ast.Ident: if mapFuncs[fn.Name] { hit = fn.Name + "()" }
                        case *ast.SelectorExpr: if mapFuncs[fn.Sel.Name] { hit = fn.Sel.Name + "()" }
                    }
            }
            if hit != "" {
                draws, stops, appends := false, false, false
                ast.Inspect(r.Body, func(m ast.Node) bool {
                    switch b := m.(type) {
                        case *ast.SelectorExpr: if id, ok := b.X.(*ast.Ident); ok && id.Name == "rand" { draws = true }
                        case *ast.ReturnStmt: stops = true
                        case *ast.CallExpr: if id, ok := b.Fun.(*ast.Ident); ok && id.Name == "append" { appends = true }
                        case *ast.BranchStmt: if b.Tok == token.BREAK && b.Label == nil { stops = true }
                        case *ast.FuncLit: return false
                        case *ast.RangeStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
                            if m != ast.Node(r.Body) { // breaks inside inner loops/switches belong to them
                                ast.Inspect(m, func(k ast.Node) bool {
                                    if sel, ok := k.(*ast.SelectorExpr); ok { if id, ok := sel.X.(*ast.Ident); ok && id.Name == "rand" { draws = true } }
                                    if _, ok := k.(*ast.ReturnStmt); ok { stops = true }
                                    return true
                                })
                                return false
                            }
                    }
                    return true
                })
                tag := ""
                if draws { tag += " DRAWS" }
                if stops { tag += " STOPS" }
                if appends { tag += " APPENDS" }
                out = append(out, fmt.Sprintf("%v:%v %v%v", filepath.ToSlash(rel), fset.Position(r.Pos()).Line, hit, tag))
            }
            return true
        })
    }
    sort.Strings(out)
    for _, line := range out { fmt.Println(line) }
}
