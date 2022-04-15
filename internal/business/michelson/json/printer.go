package json

import (
	"encoding/json"
	"fmt"

	"github.com/romarq/visualtez-testing/internal/business/michelson/ast"
)

func Print(n ast.Node, prefix string, indent string) (json.RawMessage, error) {
	return json.MarshalIndent(translateAST(n), prefix, indent)
}

func translateAST(n ast.Node) interface{} {
	var obj interface{}

	switch node := n.(type) {
	case ast.Bytes:
		obj = MichelsonJSON{Bytes: node.Value[2:]}
	case ast.Int:
		obj = MichelsonJSON{Int: fmt.Sprint(node.Value)}
	case ast.String:
		obj = MichelsonJSON{String: node.Value}
	case ast.Prim:
		prim := MichelsonJSON{
			Prim: node.Prim,
		}
		for _, el := range node.Annotations {
			prim.Annots = append(prim.Annots, el.Value)
		}
		for _, el := range node.Arguments {
			prim.Args = append(prim.Args, translateAST(el))
		}
		obj = prim
	case ast.Sequence:
		sequence := make([]interface{}, 0)
		for _, el := range node.Elements {
			sequence = append(sequence, translateAST(el))
		}
		obj = sequence
	}

	return obj
}
