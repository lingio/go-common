package main

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
)

type bucketParams struct {
	Objects []bucketObjects
	Service string
}

type bucketObjects struct {
	Name   string
	Schema typeDef
}

type typeDef struct {
	Type, Format string
	ArrayType    *typeDef
	Properties   []property
}

type property struct {
	JsonFieldName string
	Schema        typeDef
	Required      bool
}

var typeAliases = make(map[string]string)
var structDefs = make(map[string]*ast.StructType)

var timeSchema = typeDef{
	Type:   "string",
	Format: "datetime",
}

func generateBucketBrowserSwagger(projdir string, spec StorageSpec) {
	var typeNames []string
	for _, b := range spec.Buckets {
		typeNames = append(typeNames, b.DbTypeName)
	}

	loadModelsFromDir(path.Join(projdir, "storage"), ModelFileFilter)
	loadModelsFromDir(path.Join(projdir, "models"), nil)

	tmplData := bucketParams{
		Objects: []bucketObjects{},
		Service: spec.ServiceName,
	}

	for _, dbTypeName := range typeNames {
		var typdef typeDef

		if def, ok := structDefs[dbTypeName]; ok {
			typdef = traverseStruct(def)
		} else if alias, ok := typeAliases[dbTypeName]; ok {
			if alias != "[]byte" {
				log.Fatalln("cannot handle non-[]byte top level alias")
			}
			typdef = swaggerSchemaForType(alias)
		} else {
			log.Fatalln("Cannot find definition for model:", dbTypeName)
		}

		tmplData.Objects = append(tmplData.Objects, bucketObjects{
			Name:   dbTypeName,
			Schema: typdef,
		})
	}

	tpl := template.Must(template.ParseFiles("./tmpl/bucket_browser.tmpl"))
	if err := tpl.Execute(os.Stdout, tmplData); err != nil {
		log.Fatalln(err)
	}
	l := log.New(os.Stdout, "        ", 0)
	for _, obj := range tmplData.Objects {
		l.Printf("%s:\n", obj.Name)
		l.SetPrefix(l.Prefix() + "  ")
		outputObject(l, obj.Schema)
		l.SetPrefix(l.Prefix()[2:])
		l.Println()
	}
}

func outputObject(l *log.Logger, def typeDef) {
	l.Printf("type: %s\n", def.Type)
	if def.Format != "" {
		l.Printf("format: %s\n", def.Format)
	}
	if len(def.Properties) > 0 {
		l.Printf("properties:\n")
		l.SetPrefix(l.Prefix() + "  ")
		for _, prop := range def.Properties {
			l.Printf("%s:\n", prop.JsonFieldName)
			l.SetPrefix(l.Prefix() + "  ")
			outputObject(l, prop.Schema)
			l.SetPrefix(l.Prefix()[2:])
		}
		l.SetPrefix(l.Prefix()[2:])

		l.Printf("required:\n")
		l.SetPrefix(l.Prefix() + "  ")
		for _, prop := range def.Properties {
			if prop.Required {
				l.Printf("- %s\n", prop.JsonFieldName)
			}
		}
		l.SetPrefix(l.Prefix()[2:])
	}
	if def.ArrayType != nil {
		l.Printf("items:\n")
		l.SetPrefix(l.Prefix() + "  ")
		outputObject(l, *def.ArrayType)
		l.SetPrefix(l.Prefix()[2:])
	}
}

func traverseStruct(def *ast.StructType) typeDef {
	typdef := typeDef{
		Type: "object",
	}

	for _, field := range def.Fields.List {
		var schema typeDef
		required := true

		if ss, ok := field.Type.(*ast.StarExpr); ok {
			required = false
			schema = swaggerSchemaForExpr(ss.X)
		} else {
			schema = swaggerSchemaForExpr(field.Type)
		}

		for _, name := range field.Names {
			typdef.Properties = append(typdef.Properties, property{
				JsonFieldName: jsonFieldName(name.Name, field.Tag),
				Required:      required,
				Schema:        schema,
			})
		}
	}

	return typdef
}

func swaggerSchemaForExpr(expr ast.Expr) typeDef {
	if i, ok := expr.(*ast.Ident); ok {
		return swaggerSchemaForType(i.Name)
	} else if _, ok := expr.(*ast.StarExpr); ok {
		// should be handled further up in traverseStruct
		log.Fatalln("cannot handle optional expr")
	} else if ss, ok := expr.(*ast.SelectorExpr); ok {
		if pkg, ok := ss.X.(*ast.Ident); ok {
			if pkg.Name == "time" && ss.Sel.Name == "Time" {
				return timeSchema
			}
			log.Fatalf("need manual typedef for: %s.%s\n", pkg.Name, ss.Sel.Name)
		} else {
			log.Fatalf("selected unknown type: %s (%T)\n", ss.Sel.Name, ss.X)

		}
	} else if arr, ok := expr.(*ast.ArrayType); ok {
		// the magic of go save us from undefined behaviour!
		eltype := swaggerSchemaForExpr(arr.Elt)
		return typeDef{
			Type:      "array",
			ArrayType: &eltype,
		}
	} else {
		log.Fatalf("unknown expr: %T\n", expr)
	}

	// unreachable!
	return typeDef{}
}

func swaggerSchemaForType(typ string) typeDef {
	if _, ok := basicTypes[typ]; ok {
		return swaggerSchemaForBasicType(typ)
	} else if def, ok := structDefs[typ]; ok {
		return traverseStruct(def)
	} else if alias, ok := typeAliases[typ]; ok {
		return swaggerSchemaForType(alias)
	} else if typ == "[]byte" {
		return typeDef{
			Type:   "string",
			Format: "byte",
		}
	} else {
		log.Fatalln(typ, "is a not basic type and could not find struct def")
	}
	return typeDef{}
}

func jsonFieldName(fieldName string, tag *ast.BasicLit) string {
	if tag == nil {
		return fieldName
	}

	tagstr := tag.Value
	a := strings.Index(tagstr, "json:\"")
	if a == -1 {
		return fieldName
	}
	b := strings.Index(tagstr[a+6+1:], "\"")
	if b == -1 {
		return fieldName
	}

	switch tag := tagstr[a+6 : b+6+2]; tag {
	case "-":
	case "":
		return fieldName
	default:
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = fieldName
		}
		return name
	}

	log.Fatalf("unreachable !\n")
	return ""
}

func loadModelsFromDir(dirname string, fileFilter func(fs.FileInfo) bool) {
	fileSet := token.NewFileSet()
	pkgs, err := goparser.ParseDir(fileSet, dirname, fileFilter, goparser.ParseComments)
	if err != nil {
		log.Fatalf("Can not parse dir: %s: %v\n", dirname, err)
	}

	for pkgname, astpkg := range pkgs {
		log.Println("scanning pkg", pkgname)
		for _, astfile := range astpkg.Files {
			log.Println("scanning file", astfile.Name)
			for _, decl := range astfile.Decls {
				switch ds := decl.(type) {
				case *ast.GenDecl:
					if ds.Tok == token.TYPE {
						for _, spec := range ds.Specs {
							switch spec := spec.(type) {
							case *ast.TypeSpec:
								switch typespec := spec.Type.(type) {
								case *ast.StructType:
									structDefs[spec.Name.Name] = typespec
								case *ast.Ident:
									typeAliases[spec.Name.Name] = typespec.Name
								case *ast.ArrayType:
									switch eltspec := typespec.Elt.(type) {
									case *ast.Ident:
										// make []byte a special case for simplicity's sake
										if eltspec.Name == "byte" {
											typeAliases[spec.Name.Name] = "[]byte"
											fmt.Println(spec.Name.Name)
										} else {
											fmt.Printf("cannot handle array %s element type: %T\n", spec.Name.Name, typespec)
										}
									default:
										fmt.Printf("unknown array %s element type: %T\n", spec.Name.Name, typespec)
									}
								default:
									fmt.Printf("unknown spec type for %s: %T\n", spec.Name.Name, typespec)

								}
							}
						}
					}
				}
			}
		}
	}
}

func ModelFileFilter(info os.FileInfo) bool {
	name := info.Name()
	return name == "model.go" || name == "model.gen.go"
}

// copied from bulitins.go
func swaggerSchemaForBasicType(typeName string) typeDef {
	switch typeName {
	case "bool":
		return typeDef{Type: "boolean", Format: ""}
	case "byte":
		return typeDef{Type: "integer", Format: "uint8"}
	case "complex128", "complex64":
		log.Fatalln("cannot marshal complex64/complex128")
	case "error":
		return typeDef{Type: "string", Format: ""}
	case "float32":
		return typeDef{Type: "number", Format: "float"}
	case "float64":
		return typeDef{Type: "number", Format: "double"}
	case "int":
		return typeDef{Type: "integer", Format: "int64"}
	case "int16":
		return typeDef{Type: "integer", Format: "int16"}
	case "int32":
		return typeDef{Type: "integer", Format: "int32"}
	case "int64":
		return typeDef{Type: "integer", Format: "int64"}
	case "int8":
		return typeDef{Type: "integer", Format: "int8"}
	case "rune":
		return typeDef{Type: "integer", Format: "int32"}
	case "string":
		return typeDef{Type: "string", Format: ""}
	case "uint":
		return typeDef{Type: "integer", Format: "uint64"}
	case "uint16":
		return typeDef{Type: "integer", Format: "uint16"}
	case "uint32":
		return typeDef{Type: "integer", Format: "uint32"}
	case "uint64":
		return typeDef{Type: "integer", Format: "uint64"}
	case "uint8":
		return typeDef{Type: "integer", Format: "uint8"}
	case "uintptr":
		return typeDef{Type: "integer", Format: "uint64"}
	default:
		log.Fatalf("unsupported type %q\n", typeName)
	}
	return typeDef{}
}

// copied from bulitins.go
var basicTypes = map[string]bool{
	"bool":       true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"float32":    true,
	"float64":    true,
	"string":     true,
	"complex64":  true,
	"complex128": true,
	"byte":       true,
	"rune":       true,
	"uintptr":    true,
	"error":      true,
	"Time":       true,
	"file":       true,
}
