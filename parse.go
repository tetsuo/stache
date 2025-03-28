package stache

import (
	"bytes"
	"io"
	"log"

	"golang.org/x/net/html/atom"
)

var voidAtoms = map[atom.Atom]bool{
	atom.Area:   true,
	atom.Br:     true,
	atom.Embed:  true,
	atom.Img:    true,
	atom.Input:  true,
	atom.Wbr:    true,
	atom.Col:    true,
	atom.Hr:     true,
	atom.Link:   true,
	atom.Track:  true,
	atom.Source: true,
} // meta, base, param, keygen not included

type insertionMode func(*parser) bool

type parser struct {
	z                   *Tokenizer
	oe                  nodeStack
	doc                 *Node
	im                  insertionMode
	tt                  TokenType
	controlStack        controlStack
	scopeStack          scopeStack
	hasSelfClosingToken bool
}

func initialIM(p *parser) bool {
	p.oe = append(p.oe, p.doc)
	p.im = inBodyIM
	return p.im(p)
}

func inBodyIM(p *parser) bool {
	switch p.tt {
	case TextToken:
		text := bytes.Clone(bytes.TrimSpace(p.z.Raw()))
		if len(text) == 0 {
			return true
		}
		n := &Node{
			Type: TextNode,
			Data: text,
		}
		p.oe.top().AppendChild(n)
		return true
	case StartTagToken:
		name, hasAttr := p.z.TagName()

		elem := &Node{
			Type:     ElementNode,
			Data:     name,
			DataAtom: atom.Lookup(name),
		}

		for hasAttr {
			key, val, isExpr, more := p.z.TagAttr()
			elem.Attr = append(elem.Attr, Attribute{
				Key:    key,
				Val:    val,
				IsExpr: isExpr,
			})
			hasAttr = more
		}

		p.oe.top().AppendChild(elem)

		if p.hasSelfClosingToken || voidAtoms[elem.DataAtom] {
			p.hasSelfClosingToken = false
			return true
		}

		p.oe = append(p.oe, elem)
		return true
	case EndTagToken:
		name, _ := p.z.TagName()
		for i := len(p.oe) - 1; i >= 0; i-- {
			n := p.oe[i]
			if n.DataAtom != 0 {
				if n.DataAtom == atom.Lookup(name) {
					p.oe = p.oe[:i]
					break
				}
			} else {
				if bytes.Equal(n.Data, name) {
					p.oe = p.oe[:i]
					break
				}
			}
		}
		return true
	case VariableToken:
		varName := bytes.TrimSpace(p.z.Raw())
		path := make([][]byte, len(p.scopeStack), len(p.scopeStack)+1)
		copy(path, p.scopeStack)
		path = append(path, varName)

		n := &Node{
			Type: VariableNode,
			Data: varName,
			Path: path,
		}
		p.oe.top().AppendChild(n)
		return true
	case WhenToken:
		name := bytes.Clone(bytes.TrimSpace(p.z.ControlName()))
		node := &Node{Type: WhenNode, Data: name}
		p.oe.top().AppendChild(node)
		p.oe = append(p.oe, node)
		p.controlStack.push(controlFrame{typ: WhenNode, name: node.Data})
		return true
	case UnlessToken:
		name := bytes.Clone(bytes.TrimSpace(p.z.ControlName()))
		node := &Node{Type: UnlessNode, Data: name}
		p.oe.top().AppendChild(node)
		p.oe = append(p.oe, node)
		p.controlStack.push(controlFrame{typ: UnlessNode, name: node.Data})
		return true
	case RangeToken:
		name := bytes.Clone(bytes.TrimSpace(p.z.ControlName()))
		segments := p.scopeStack.pushSegments(name)
		node := &Node{Type: RangeNode, Data: name}
		p.oe.top().AppendChild(node)
		p.oe = append(p.oe, node)
		p.controlStack.push(controlFrame{typ: RangeNode, name: name, segments: segments})
		return true
	case EndControlToken:
		name := bytes.TrimSpace(p.z.ControlName())
		top := p.controlStack.top()
		if top == nil {
			// TODO: unexpected {/...}
			return true
		}
		if !bytes.Equal(top.name, name) {
			log.Printf("mismatched control end: expected {/%s}, got {/%s}\n", top.name, name)
			return true
		}
		p.controlStack.pop()
		p.scopeStack.popN(len(top.segments))

		// pop matching node
		for i := len(p.oe) - 1; i >= 0; i-- {
			n := p.oe[i]
			if n.Type == top.typ && bytes.Equal(n.Data, name) {
				p.oe = p.oe[:i]
				break
			}
		}
		return true
	case CommentToken:
		p.doc.AppendChild(&Node{
			Type: CommentNode,
			Data: bytes.Clone(bytes.TrimSpace(p.z.Comment())),
		})
		return true
	}
	return false
}

func (p *parser) parseCurrentToken() {
	if p.tt == SelfClosingTagToken {
		p.hasSelfClosingToken = true
		p.tt = StartTagToken
	}

	consumed := false
	for !consumed {
		consumed = p.im(p)
	}
}

func (p *parser) parse() error {
	var err error
	for err != io.EOF {
		p.tt = p.z.Next()
		if p.tt == ErrorToken {
			if err := p.z.Err(); err != nil && err != io.EOF {
				return err
			}
			break
		}
		p.parseCurrentToken()
	}
	return nil
}

func Parse(r io.Reader) (*Node, error) {
	p := &parser{
		z:  NewTokenizer(r),
		im: initialIM,
		doc: &Node{
			Type: ComponentNode,
		},
	}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.doc, nil
}
