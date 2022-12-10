package js

import (
	"github.com/stretchr/testify/require"
	"path"
	"path/filepath"
	"testing"
)

const testFolder = "parser_test"

func TestParser_Parse(t *testing.T) {
	a := require.New(t)
	id := path.Join(testFolder, t.Name()+".js")
	absPath, err := filepath.Abs(id)
	a.NoError(err)

	node, err := Parser.Parse(id)
	a.NoError(err)
	a.Equal(node.Id, absPath)
	a.Equal(node.Data.dirname, path.Dir(absPath))
	a.Equal(node.Data.content, []byte("console.log(\"hello world!\")\n"))
}

func TestParser_Deps(t *testing.T) {
	a := require.New(t)
	id := path.Join(testFolder, t.Name()+".js")

	node, err := Parser.Parse(id)
	a.NoError(err)
	deps := Parser.Deps(node)
	a.Equal(len(deps), 9)
}

func TestParser_Deps_imports(t *testing.T) {
	a := require.New(t)
	id := path.Join(testFolder, t.Name()+".js")
	node, err := Parser.Parse(id)
	a.NoError(err)

	deps := Parser.Deps(node)
	a.Equal(len(deps), 2)
	for _, dep := range deps {
		node, err := Parser.Parse(dep)
		a.NoError(err)
		a.Equal(node.Data.content, []byte("console.log(\"hello world!\")\n"))
	}
}
