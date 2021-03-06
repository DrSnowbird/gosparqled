package autocompletion

import (
    "testing"
    "bytes"
)

// Add a triple pattern to the recommendation query
func (td *Scope) add(s string, p string, o string) {
    td.Tps = append(td.Tps, triplePattern{ S : s, P : p, O : o })
}

// Same as add but specify that the object is never used as a subject
func (td *Scope) addLeaf(s string, p string, o string) {
    td.Tps = append(td.Tps, triplePattern{ S : s, P : p, O : o, Leaf : true })
}

// Gets the RecommendationQuery from query and compare it against the expected one
func parse(t *testing.T, query string, expected *Scope, rType Type) *Sparql {
    s := &Sparql{ Buffer : query, Scope : NewScope() }
    s.Init()
    parseWithSparql(t, s, expected, rType)
    return s
}

// Like parse but pass a custom query recommendation template
func parseWithTemplate(t *testing.T, query string, tmpl string, expected *Scope, rType Type) *Sparql {
    s := &Sparql{ Buffer : query, Scope : NewScopeWithTemplate(tmpl) }
    s.Init()
    parseWithSparql(t, s, expected, rType)
    return s
}

// Parse a SPARQL query which is expected to throw an error
func parseInvalid(t *testing.T, query string) {
    s := &Sparql{ Buffer : query, Scope : NewScope() }
    s.Init()
    if err := s.Parse(); err == nil {
        t.Error("Expected parse error!")
    }
}

// Like parse but pass the Sparql object as argument instead
func parseWithSparql(t *testing.T, s *Sparql, expected *Scope, rType Type) {
    if err := s.Parse(); err != nil {
        t.Errorf("Failed to parse query\n%v", err)
    }
    s.Execute()
    actual := s.RecommendationQuery()
    var out bytes.Buffer
    s.template.Execute(&out, expected)
    expectedString := out.String()
    if actual != expectedString {
        t.Errorf("Expected %v\nbut got %v\n", expectedString, actual)
    }
    aType := s.RecommendationType()
    if rType != aType {
        t.Errorf("Expected Recommendation type to be [%v] but got [%v]\n", rType, aType)
    }
}

func TestPofSubject(t *testing.T) {
    tmpl := "{{.PofSubject}}"
    td := NewScope()
    td.addLeaf("?s", "a", "?POF")
    td.add("?s", "?p", "?o")
    parseWithTemplate(t, `
        SELECT *
        WHERE {
            ?s a < ; ?p ?o .
            ?o <aaa> <bbb>
        }
    `, tmpl, td, CLASS)
}

func TestLeaves1(t *testing.T) {
    tmpl := `{{range .Tps}}
                {{if eq .Leaf true}}
                    {{.O}}
                {{end}}
             {{end}}`
    td := NewScope()
    td.add("?s", "a", "?o")
    td.addLeaf("?s", "<p1>", "?o1")
    td.addLeaf("?o", "a", ":Person")
    td.addLeaf("?o", "?POF", "?FillVar")
    parseWithTemplate(t, `
        SELECT *
        WHERE {
            ?s a ?o; <p1> ?o1 .
            ?o a :Person; <
        }
    `, tmpl, td, PREDICATE)
}

func TestLeaves2(t *testing.T) {
    tmpl := `{{range .Tps}}
                {{if eq .Leaf true}}
                    {{.O}}
                {{end}}
             {{end}}`
    td := NewScope()
    td.add("?s", "a", "<aaa>")
    td.addLeaf("?s", "<p1>", "?o1")
    td.addLeaf("<aaa>", "a", ":Person")
    td.addLeaf("<aaa>", "?POF", "?FillVar")
    parseWithTemplate(t, `
        SELECT *
        WHERE {
            ?s a <aaa>; <p1> ?o1 .
            <aaa> a :Person; <
        }
    `, tmpl, td, PREDICATE)
}

func TestMissingPredicateOrObject(t *testing.T) {
    parseInvalid(t, `
        SELECT *
        WHERE {
            ?s <aaa>; <
        }
        `)
}

func TestObjectPof(t *testing.T) {
    td := NewScope()
    td.add("?s", "<aaa>", "?POF")
    parse(t, `
        SELECT *
        WHERE {
            ?s <aaa> <
        }
        `, td, OBJECT)
}

func TestTwoTypes(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", ":Person")
    td.add("?s", "a", "?POF")
    parse(t, `
        SELECT *
        WHERE {
            ?s a :Person; a <
        }
        `, td, CLASS)
}

func TestFilter(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", "?POF")
    td.add("?s", "<name>", "?name")
    parse(t, `# Test comment
        SELECT *
        WHERE {
            ?s a <
            FILTER (lang(?name) = "en")
            ?s <name> ?name
        }
        `, td, CLASS)
}

func TestComment1(t *testing.T) {
    td := NewScope()
    td.add("?POF", "?p", "?o")
    parse(t, `# Test comment
        SELECT *
        WHERE {
            <   
            # blabla
            ?p ?o 
        }
        `, td, SUBJECT)
}

func TestComment2(t *testing.T) {
    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    parse(t, `# Test comment
        SELECT *
        WHERE {
            ?s # blabla
                < 
        }
        `, td, PREDICATE)
}

func TestComment3(t *testing.T) {
    td := NewScope()
    td.add("?s", "?p", "?POF")
    parse(t, `# Test comment
        SELECT *
        WHERE {
            ?s ?p # blabla
                < 
        }
        `, td, OBJECT)
}

func TestComment4(t *testing.T) {
    query := `
        SELECT *
        WHERE {
            ?a ?b ?c .
            ?s <
            # test
        }
    `
    scope := NewScope()
    s := &Sparql{ Buffer : query, Scope : scope }
    s.Init()

    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    parseWithSparql(t, s, td, PREDICATE)
    Reset(s)
    parseWithSparql(t, s, td, PREDICATE)
}

func TestReset(t *testing.T) {
    query := `
        SELECT *
        WHERE {
            ?s < 
        }
    `
    scope := NewScope()
    s := &Sparql{ Buffer : query, Scope : scope }
    s.Init()

    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    parseWithSparql(t, s, td, PREDICATE)
    Reset(s)
    parseWithSparql(t, s, td, PREDICATE)
}

func TestPrefix(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", "a:bbb")
    td.add("?s", "?POF", "?FillVar")
    td.Prefixes = map[string]string{ "a" : "aaa" }
    parse(t, `
        PREFIX a: <aaa>
        SELECT *
        WHERE {
            ?s a a:bbb; < 
        }
        `, td, PREDICATE)
}

func TestPrefixRecommendation1(t *testing.T) {
    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    td.Prefix = "aaa"
    td.Prefixes = map[string]string{ "a" : "aaa" }
    parse(t, `
        PREFIX a: <aaa>
        SELECT *
        WHERE {
            ?s a:< 
        }
        `, td, PREDICATE)
}

func TestPrefixRecommendation2(t *testing.T) {
    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    td.Prefix = "aaa"
    td.Prefixes = map[string]string{ "" : "aaa" }
    parse(t, `
        PREFIX : <aaa>
        SELECT *
        WHERE {
            ?s :< 
        }
        `, td, PREDICATE)
}

func TestPrefixRecommendation3(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", "?POF")
    td.Prefix = "bbb"
    td.Prefixes = map[string]string{ "a" : "aaa", "b" : "bbb", "c" : "ccc" }
    parse(t, `
        PREFIX a: <aaa>
        PREFIX b: <bbb>
        PREFIX c: <ccc>
        SELECT *
        WHERE {
            ?s a b:< 
        }
        `, td, CLASS)
}

func TestPrefixRecommendation4(t *testing.T) {
    td := NewScope()
    td.add("?s", ":bbb", "?o")
    td.add("?s", "?POF", "?FillVar")
    td.Prefixes = map[string]string{ "" : "aaa" }
    parse(t, `
        PREFIX : <aaa>
        SELECT * WHERE {
          ?s :bbb ?o; <
        } 
        LIMIT 10
        `, td, PREDICATE)
}

func TestLimitOffset(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", "?POF")
    parse(t, `
        SELECT * WHERE {
          ?s a <
        }
        offset 10
        LIMIT 10
        `, td, CLASS)
}

func TestPath1(t *testing.T) {
    td := NewScope()
    td.add("?sFillVar2", "?POF3", "?FillVar")
    td.add("?s", "?POF1", "?sFillVar1")
    td.add("?sFillVar1", "?POF2", "?sFillVar2")
    td.Pof = pathPof(3)
    parse(t, `
        SELECT *
        WHERE {
          ?s 3/< 
        }
        `, td, PATH)
}

func TestPath2(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", "<aaa>")
    td.add("?sFillVar2", "?POF3", "?FillVar")
    td.add("?s", "?POF1", "?sFillVar1")
    td.add("?sFillVar1", "?POF2", "?sFillVar2")
    td.Pof = pathPof(3)
    parse(t, `
        SELECT *
        WHERE {
          ?s a <aaa>; 3/< 
        }
        `, td, PATH)
}

func TestEval1(t *testing.T) {
    td := NewScope()
    td.add("?v0", "a", "?POF")
    td.add("?v1", "<http://dbpedia.org/ontology/developer>", "?v0")
    td.add("?v1", "a", "<http://dbpedia.org/ontology/Software>")
    parse(t, `
        SELECT *
        WHERE {
          ?v0 a  <  .
          ?v1 <http://dbpedia.org/ontology/developer> ?v0 .
          ?v1 a <http://dbpedia.org/ontology/Software> .
        }
        `, td, CLASS)
}

func TestEval2(t *testing.T) {
    td := NewScope()
    td.add("?v0", "a", "?POF")
    td.add("?v0", "<http://dbpedia.org/ontology/director>", "?v1")
    td.add("?v0", "<http://xmlns.com/foaf/0.1/name>", "?v2")
    td.add("?v0", "<http://dbpedia.org/property/imdbId>", "?v3")
    td.add("?v1", "<http://dbpedia.org/property/dateOfBirth>", "?v4")
    parse(t, `
        SELECT *
        WHERE {
            ?v0 a  <  .
            ?v0 <http://dbpedia.org/ontology/director> ?v1 .
            ?v0 <http://xmlns.com/foaf/0.1/name> ?v2 .
            ?v0 <http://dbpedia.org/property/imdbId> ?v3 .
            ?v1 <http://dbpedia.org/property/dateOfBirth> ?v4 .
        }
        `, td, CLASS)
}

func TestEval3(t *testing.T) {
    td := NewScope()
    td.add("?v0", "a", "?POF")
    td.add("?v0", "<http://dbpedia.org/ontology/birthdate>", "?v1")
    td.add("?v0", "<http://xmlns.com/foaf/0.1/name>", "?v2")
    td.add("?v0", "<http://dbpedia.org/property/abstract>", "?v3")
    parse(t, `
        SELECT *
        WHERE {
            ?v0 a  <  ;<http://dbpedia.org/ontology/birthdate> ?v1 ;<http://xmlns.com/foaf/0.1/name> ?v2 ;<http://dbpedia.org/property/abstract> ?v3 .
        }
        `, td, CLASS)
}

func TestKeyword1(t *testing.T) {
    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    td.Keyword = "test"
    parse(t, `
        SELECT * WHERE {
          ?s test< 
        }
        LIMIT 10
    `, td, PREDICATE)
}

func TestKeyword2(t *testing.T) {
    td := NewScope()
    td.add("?s", "?p", "?POF")
    td.Keyword = "test"
    parse(t, `
        SELECT * WHERE {
          ?s ?p test< 
        }
        LIMIT 10
    `, td, OBJECT)
}

func TestKeyword3(t *testing.T) {
    td := NewScope()
    td.add("?s", "a", "?POF")
    td.Keyword = "Person-1"
    parse(t, `
        SELECT * WHERE {
          ?s a Person-1< 
        }
        LIMIT 10
    `, td, CLASS)
}

func TestEditor(t *testing.T) {
    td := NewScope()
    td.add("?sub", "a", "<http://schema.org/MusicGroup>")
    td.add("?sub", "?POF", "?FillVar")
    parse(t, `
        SELECT * WHERE {
          ?sub a <http://schema.org/MusicGroup> .
          ?sub < 
        }
        LIMIT 10
    `, td, PREDICATE)
}

func TestSubject(t *testing.T) {
    td := NewScope()
    td.add("?POF", "?p", "?o1")
    td.add("?POF", "a", "?o2")
    parse(t, `
        select * {
            < ?p ?o1; a ?o2 .
            ?o ?op ?oo .
            ?a ?b ?c
        }
    `, td, SUBJECT)
}

func TestPredicate(t *testing.T) {
    td := NewScope()
    td.add("?s", "?POF", "?FillVar")
    td.add("?s", "a", "?o")
    td.add("?o", "?op", "?oo")
    parse(t, `
        select * {
            ?s < ; a ?o .
            ?o ?op ?oo .
            ?a ?b ?c
        }
    `, td, PREDICATE)
}

func TestObject(t *testing.T) {
    td := NewScope()
    td.add("?s", "?p", "?POF")
    td.add("?s", "a", "?o")
    td.add("?o", "?op", "?oo")
    parse(t, `
        select * {
            ?s ?p < ; a ?o .
            ?o ?op ?oo .
            ?a ?b ?c
        }
    `, td, OBJECT)
}

func TestTerms(t *testing.T) {
    td := NewScope()
    td.add("?s", "<p1>", "?POF")
    td.add("?s", "a", "?o")
    td.add("?o", "?p", "\"test\"")
    parse(t, `
        select * {
            ?s <p1> < ; a ?o .
            ?o ?p "test" .
        }
    `, td, OBJECT)
}

func TestOptional1(t *testing.T) {
    td := NewScope()
    td.add("?s", "<p1>", "?o")
    td.add("?o", "?POF", "?FillVar")
    parse(t, `
        select * {
            ?s <p1> ?o .
            OPTIONAL { ?o < } .
        }
    `, td, PREDICATE)
}

func TestOptional2(t *testing.T) {
    td := NewScope()
    td.add("?o", "?POF", "?FillVar")
    td.add("?s", "<p1>", "?o")
    parse(t, `
        select * {
            OPTIONAL { ?o < } .
            ?s <p1> ?o .
        }
    `, td, PREDICATE)
}

func TestOptional3(t *testing.T) {
    td := NewScope()
    td.add("?s", "<p1>", "?o")
    td.add("?o", "?POF", "?FillVar")
    td.add("?s", "<p1>", "?o")
    parse(t, `
        select * {
            ?s <p1> ?o .
            OPTIONAL { ?o < } .
            ?s <p1> ?o .
        }
    `, td, PREDICATE)
}

