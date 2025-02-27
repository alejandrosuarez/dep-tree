package language

import (
	"github.com/elliotchance/orderedmap/v2"
	"github.com/gabotechs/dep-tree/internal/dep_tree"
	"github.com/gabotechs/dep-tree/internal/graph"
	"github.com/gabotechs/dep-tree/internal/utils"
)

type Node = graph.Node[FileInfo]
type Graph = graph.Graph[FileInfo]
type NodeParser = dep_tree.NodeParser[FileInfo]
type NodeParserBuilder = dep_tree.NodeParserBuilder[FileInfo]
type DisplayResult = dep_tree.DisplayResult

type FileInfo struct {
	Loc  int
	Size int
}

type CodeFile interface {
	Loc() int
	Size() int
}

type Language[F CodeFile] interface {
	// ParseFile receives an absolute file path and returns F, where F is the specific file implementation
	//  defined by the language. This file object F will be used as input for parsing imports and exports.
	ParseFile(path string) (*F, error)
	// ParseImports receives the file F parsed by the ParseFile method and gathers the imports that the file
	//  F contains.
	ParseImports(file *F) (*ImportsResult, error)
	// ParseExports receives the file F parsed by the ParseFile method and gathers the exports that the file
	//  F contains.
	ParseExports(file *F) (*ExportsEntries, error)
	// Display takes an absolute path to a file and displays it nicely.
	Display(path string) DisplayResult
}

type Parser[F CodeFile] struct {
	lang               Language[F]
	unwrapProxyExports bool
	exclude            []string
	// cache
	fileCache    map[string]*F
	importsCache map[string]*ImportsResult
	exportsCache map[string]*ExportsResult
}

var _ NodeParser = &Parser[CodeFile]{}

type Config interface {
	UnwrapProxyExports() bool
	IgnoreFiles() []string
}

type Builder[F CodeFile, C any] func(C) (Language[F], error)

func ParserBuilder[F CodeFile, C any](languageBuilder Builder[F, C], langCfg C, generalCfg Config) NodeParserBuilder {
	fileCache := map[string]*F{}
	importsCache := map[string]*ImportsResult{}
	exportsCache := map[string]*ExportsResult{}
	return func(files []string) (NodeParser, error) {
		lang, err := languageBuilder(langCfg)
		if err != nil {
			return nil, err
		}

		parser := &Parser[F]{
			lang:               lang,
			unwrapProxyExports: true,
			fileCache:          fileCache,
			importsCache:       importsCache,
			exportsCache:       exportsCache,
		}
		if generalCfg != nil {
			parser.unwrapProxyExports = generalCfg.UnwrapProxyExports()
			parser.exclude = generalCfg.IgnoreFiles()
		}
		return parser, err
	}
}

func (p *Parser[F]) shouldExclude(path string) bool {
	for _, exclusion := range p.exclude {
		if ok, _ := utils.GlobstarMatch(exclusion, path); ok {
			return true
		}
	}
	return false
}

func (p *Parser[F]) Node(id string) (*Node, error) {
	return graph.MakeNode(id, FileInfo{}), nil
}

func (p *Parser[F]) updateNodeInfo(n *Node) error {
	file, err := p.parseFile(n.Id)
	if err != nil {
		return err
	}
	n.Data.Size = (*file).Size()
	n.Data.Loc = (*file).Loc()
	return nil
}

func (p *Parser[F]) Deps(n *Node) ([]*Node, error) {
	_ = p.updateNodeInfo(n)
	imports, err := p.gatherImportsFromFile(n.Id)
	if err != nil {
		return nil, err
	}
	n.AddErrors(imports.Errors...)

	// Some exports might be re-exporting symbols from other files, we consider
	// those as if they where normal imports.
	//
	// TODO: if exports are parsed as imports, they might say that that a name is being
	//  imported from a path when it's actually not available.
	//  ex:
	//   index.ts -> import { foo } from 'foo.ts'
	//   foo.ts   -> import { bar as foo } from 'bar.ts'
	//   bar.ts   -> export { bar }
	//  If unwrappedExports is true, this will say that `foo` is exported from `bar.ts`, which
	//  technically is true, but it's not true to say that `foo` is imported from `bar.ts`.
	//  It's more accurate to say that `bar` is imported from `bar.ts`, even if the alias is `foo`.
	//  Instead we never unwrap export to avoid this.
	exports, err := p.parseExports(n.Id, false, nil)
	if err != nil {
		return nil, err
	}
	n.AddErrors(exports.Errors...)
	for el := exports.Exports.Front(); el != nil; el = el.Next() {
		if el.Value != n.Id {
			imports.Imports = append(imports.Imports, ImportEntry{
				Names: []string{el.Key},
				Path:  el.Value,
			})
		}
	}

	resolvedImports := orderedmap.NewOrderedMap[string, bool]()

	// Imported names might not necessarily be declared in the path that is being imported, they might be declared in
	// a different file, we want that file. Ex: foo.ts -> utils/index.ts -> utils/sum.ts. If unwrapProxyExports is
	// set to true, we must trace those exports back.
	for _, importEntry := range imports.Imports {
		if !p.unwrapProxyExports {
			resolvedImports.Set(importEntry.Path, true)
			continue
		}

		// NOTE: at this point p.unwrapProxyExports is always true.
		exports, err = p.parseExports(importEntry.Path, p.unwrapProxyExports, nil)
		if err != nil {
			return nil, err
		}
		n.AddErrors(exports.Errors...)
		if importEntry.All {
			// If all imported, then dump every path in the resolved imports.
			for el := exports.Exports.Front(); el != nil; el = el.Next() {
				resolvedImports.Set(el.Value, true)
			}
			continue
		} else if len(importEntry.Names) == 0 {
			resolvedImports.Set(importEntry.Path, true)
		}
		for _, name := range importEntry.Names {
			if exportPath, ok := exports.Exports.Get(name); ok {
				resolvedImports.Set(exportPath, true)
			} else {
				// TODO: this is not retro-compatible, do it in a different PR.
				// n.AddErrors(fmt.Errorf("name %s is imported by %s but not exported by %s", name, n.Id, importEntry.Id)).
			}
		}
	}

	deps := make([]*Node, 0)
	for _, imported := range resolvedImports.Keys() {
		node := graph.MakeNode(imported, FileInfo{})
		if !p.shouldExclude(p.Display(node).Name) {
			deps = append(deps, node)
		}
	}
	return deps, nil
}

func (p *Parser[F]) Display(n *Node) dep_tree.DisplayResult {
	return p.lang.Display(n.Id)
}
