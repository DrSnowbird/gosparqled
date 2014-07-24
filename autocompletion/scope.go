/*
 Package autocompletion creates a SPARQL query that can be used for retrieving
 recommendations elements, e.g., predicates or classes.

 The position in the input SPARQL query for the recommendation is indicated
 by the character '<', called the "Point Of Focus". For example, the query

    SELECT * {
        ?s1 a <:Person>; < .
        ?s2 ?p ?o
    }

 will get recommendations for predicates co-occurring with a resource of
 type :Person. Only the patterns that are connected to that Point Of Focus,
 directly or not, are kept for generating the final query. For example, the
 triple pattern "?s2 ?p ?o" is removed.
*/
package autocompletion

import (
    "strings"
    "text/template"
    "bytes"
    "strconv"
)

// The kind of recommendation
type Type uint

const (
    // No recommendation
    NONE Type = iota
    // Class recommendation
    CLASS
    // Predicate recommendation
    PREDICATE
    // Path recommendation
    PATH
    // Subject recommendation
    SUBJECT
    // Object recommendation
    OBJECT
)

// A SPARQL triple pattern
type triplePattern struct {
    S, P, O string
}

// Set of triple patterns relevant for the recommendation
// A triple pattern is relevant only if it is part of the connected component
// that contains the Point Of Focus.
type Scope struct {
    triplePattern
    // The list of triple patterns
    Tps []triplePattern
    scope map[string]bool
    // The template of the SPARQL query used for retrieving recommendations
    template *template.Template
    // A keyword that the recommended item must match
    Keyword string
    // The number of properties for a path to be recommended
    // If 0, it is a direct path
    pathLength int
    // The POF expression to project in the SELECT query
    Pof string
}

// Scope struct constructor
func NewScope() *Scope {
    tmpl := `
        SELECT DISTINCT {{.Pof}}
        WHERE {
        {{range .Tps}}
            {{.S}} {{.P}} {{.O}} .
        {{end}}
        {{if .Keyword}}
            FILTER regex(?POF, "{{.Keyword}}", "i")
        {{end}}
        }
        LIMIT 10
    `
    return NewScopeWithTemplate(tmpl)
}

// Scope struct constructor with the given text template
func NewScopeWithTemplate(tmpl string) *Scope {
    scope := &Scope{ Pof : "?POF" }
    tp, _ := template.New("rec").Parse(tmpl)
    scope.template = tp
    return scope
}

// Reset the scope to prepare for a new query
func (b *Scope) Reset() {
    b.Keyword = ""
    b.pathLength = 0
    b.Pof = "?POF"
    b.Tps = b.Tps[:0]
}

// Sets the keyword that the recommended item must match
func (b *Scope) setKeyword(keyword string) {
    if len(keyword) != 0 {
        b.Keyword = keyword
    }
}

// Sets the subject of the triple pattern
func (b *Scope) setSubject(s string) {
    s = strings.TrimSpace(s)
    if (len(s) != 0) {
        b.S = s
    }
}

// Sets the predicate of the triple pattern
func (b *Scope) setPredicate(p string) {
    p = strings.TrimSpace(p)
    if (len(p) != 0) {
        b.P = p
    }
}

// Sets the object of the triple pattern
func (b *Scope) setObject(o string) {
    o = strings.TrimSpace(o)
    if (len(o) != 0) {
        b.O = o
    }
}

// Adds the current triple pattern to the Scope
func (b *Scope) addTriplePattern() {
    tp := triplePattern{ S : b.S, P : b.P, O : b.O }
    b.Tps = append(b.Tps, tp)
}

// Sets the length of the path to be recommended
func (b *Scope) setPathLength(lenght string) {
    b.pathLength, _ = strconv.Atoi(lenght)
}

// Removes triple patterns from the Scope that are not within the connected
// component that contains the Point Of Focus
func (b *Scope) trimToScope() {
    b.scope = map[string]bool{ "?POF" : true }
    size := 0
    for size != len(b.scope) {
        size = len(b.scope)
        for _, tp := range b.Tps {
            if (tp.in(b.scope)) {
                tp.addToScope(b.scope)
            }
        }
    }
    var scoped []triplePattern
    for _,tp := range b.Tps {
        if (tp.in(b.scope)) {
            scoped = append(scoped, tp)
        }
    }
    b.Tps = scoped
}

// Returns true id the triple pattern is within the scope
func (tp *triplePattern) in(scope map[string]bool) bool {
    if scope[tp.S] || scope[tp.P] || scope[tp.O] {
        return true
    }
    return false
}

// Adds the triple pattern to the scope
func (tp *triplePattern) addToScope(scope map[string]bool) {
    scope[tp.S] = true
    scope[tp.P] = true
    scope[tp.O] = true
}

// Adds the property variables for building the path to recommend of length pathLength
func (b *Scope) addIntermediatePath() {
    if b.pathLength == 0 {
        return
    }
    for ind,tp := range b.Tps {
        if tp.P == "?POF" {
            inter := tp.S
            // intermediate properties
            for i := 1; i < b.pathLength; i++ {
                inter2 := "?" + tp.S[1:] + tp.O[1:] + strconv.Itoa(i)
                tpInter := triplePattern{ S: inter, P: "?POF" + strconv.Itoa(i), O: inter2 }
                b.Tps = append(b.Tps, tpInter)
                inter = inter2
            }
            // last property
            b.Tps[ind].S = inter
            b.Tps[ind].P = "?POF" + strconv.Itoa(b.pathLength)
            b.Pof = pathPof(b.pathLength)
            break
        }
    }
}

// pathPof returns the ?POF projection expression as the concatenation
// of each intermediate variable properties
func pathPof(pathLength int) string {
    pof := "(concat("
    for i := 1; i <= pathLength; i++ {
        if i > 1 {
            pof += ", "
        }
        pof += "\"<\", ?POF" + strconv.Itoa(i) + ", \">\""
        if i < pathLength {
            pof += ", \" / \""
        }
    }
    return pof + ") as ?POF)"
}

// RecommendationType returns the kind of recommendation for the processed SPARQL query
func (b *Scope) RecommendationType() Type {
    if b.pathLength != 0 { return PATH }
    for _,tp := range b.Tps {
        if tp.P == "?POF" {
            return PREDICATE
        }
        if tp.P == "a" && tp.O == "?POF" {
            return CLASS
        }
        if tp.O == "?POF" {
            return OBJECT
        }
        if tp.S == "?POF" {
            return SUBJECT
        }
    }
    return NONE
}

// Returns the SPARQL query that can be used for retrieving recommendations
// about the Point Of Focus. The recommended items are bound to the variable
// labelled "?POF"
func (b *Scope) RecommendationQuery() string {
    b.trimToScope()
    b.addIntermediatePath()
    var out bytes.Buffer
    b.template.Execute(&out, b)
    return out.String()
}
