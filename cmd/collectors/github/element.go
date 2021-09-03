package github

import (
    "fmt"
	"strings"
	"goharvest2/pkg/tree/node"
)

// Abstraction of an API element
type Element struct {
	Name        string
	DisplayName string
	IsKey       bool
	IsLabel     bool
	IsMetric    bool
    IsNested    bool
    HasNestedKeys bool
    Elements    []*Element
}

func ParseElementTree(n *node.Node) *Element {

	var elem, child *Element

    if n.GetNameS() != "" {
        elem = ParseElement(n.GetNameS())
    } else {
        elem = ParseElement(n.GetContentS())
    }

	if len(n.GetChildren()) != 0 {
        elem.IsNested = true
		for _, c := range n.GetChildren() {
            child = ParseElementTree(c)
            elem.AddElement(child)
            if child.IsKey {
                elem.HasNestedKeys = true
            }
		}
	}
	return elem
}

func ParseElement(input string) *Element {

	e := new(Element)

	if strings.HasPrefix(input, "^^") {
		e.IsKey = true
		e.IsLabel = true
		input = strings.TrimPrefix(input, "^^")
	} else if strings.HasPrefix(input, "^") {
		e.IsLabel = true
		input = strings.TrimPrefix(input, "^")
	} else {
		e.IsMetric = true
	}

	if values := strings.Split(input, "=>"); len(values) == 2 {
		e.Name = strings.TrimSpace(values[0])
		e.DisplayName = strings.TrimSpace(values[1])
	} else {
		e.Name = strings.TrimSpace(input)
		e.DisplayName = e.Name
	}
	return e
}

func (e *Element) AddElement(c *Element) {
    e.Elements = append(e.Elements, c)
}

func (e *Element) String() string {
    return fmt.Sprintf(
		"%s (%s) key=%t label=%t metric=%t",
		e.Name, 
		e.DisplayName, 
		e.IsKey, 
		e.IsLabel, 
		e.IsMetric,
		)
}

