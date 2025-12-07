package vdom

import "testing"

func TestCreateElement(t *testing.T) {
	t.Run("basic element", func(t *testing.T) {
		node := Div()
		if node.Kind != KindElement {
			t.Errorf("Kind = %v, want KindElement", node.Kind)
		}
		if node.Tag != "div" {
			t.Errorf("Tag = %v, want div", node.Tag)
		}
	})

	t.Run("with class attribute", func(t *testing.T) {
		node := Div(Class("card"))
		if node.Props["class"] != "card" {
			t.Errorf("class = %v, want card", node.Props["class"])
		}
	})

	t.Run("with multiple attributes", func(t *testing.T) {
		node := Div(Class("card"), ID("main"))
		if node.Props["class"] != "card" {
			t.Errorf("class = %v, want card", node.Props["class"])
		}
		if node.Props["id"] != "main" {
			t.Errorf("id = %v, want main", node.Props["id"])
		}
	})

	t.Run("with child node", func(t *testing.T) {
		node := Div(P(Text("Hello")))
		if len(node.Children) != 1 {
			t.Fatalf("Children len = %v, want 1", len(node.Children))
		}
		if node.Children[0].Tag != "p" {
			t.Errorf("Child tag = %v, want p", node.Children[0].Tag)
		}
	})

	t.Run("with multiple children", func(t *testing.T) {
		node := Div(H1(Text("Title")), P(Text("Content")))
		if len(node.Children) != 2 {
			t.Fatalf("Children len = %v, want 2", len(node.Children))
		}
	})

	t.Run("with string shorthand", func(t *testing.T) {
		node := Div("Hello")
		if len(node.Children) != 1 {
			t.Fatalf("Children len = %v, want 1", len(node.Children))
		}
		if node.Children[0].Kind != KindText {
			t.Errorf("Child kind = %v, want KindText", node.Children[0].Kind)
		}
		if node.Children[0].Text != "Hello" {
			t.Errorf("Child text = %v, want Hello", node.Children[0].Text)
		}
	})

	t.Run("with nil ignored", func(t *testing.T) {
		node := Div(nil, Class("test"), nil)
		if node.Props["class"] != "test" {
			t.Errorf("class = %v, want test", node.Props["class"])
		}
		if len(node.Children) != 0 {
			t.Errorf("Children len = %v, want 0", len(node.Children))
		}
	})

	t.Run("with event handler", func(t *testing.T) {
		handler := func() {}
		node := Button(OnClick(handler))
		if node.Props["onclick"] == nil {
			t.Error("onclick handler not set")
		}
	})

	t.Run("with slice of children", func(t *testing.T) {
		children := []*VNode{Li(Text("A")), Li(Text("B"))}
		node := Ul(children)
		if len(node.Children) != 2 {
			t.Fatalf("Children len = %v, want 2", len(node.Children))
		}
	})

	t.Run("with slice containing nil", func(t *testing.T) {
		children := []*VNode{Li(Text("A")), nil, Li(Text("B"))}
		node := Ul(children)
		if len(node.Children) != 2 {
			t.Fatalf("Children len = %v, want 2 (nil filtered)", len(node.Children))
		}
	})

	t.Run("with slice of attributes", func(t *testing.T) {
		attrs := []Attr{Class("test"), ID("main")}
		node := Div(attrs)
		if node.Props["class"] != "test" {
			t.Errorf("class = %v, want test", node.Props["class"])
		}
		if node.Props["id"] != "main" {
			t.Errorf("id = %v, want main", node.Props["id"])
		}
	})

	t.Run("with key attribute", func(t *testing.T) {
		node := Div(Key("item-1"))
		if node.Key != "item-1" {
			t.Errorf("Key = %v, want item-1", node.Key)
		}
		if node.Props["key"] != "item-1" {
			t.Errorf("Props[key] = %v, want item-1", node.Props["key"])
		}
	})

	t.Run("with component", func(t *testing.T) {
		comp := Func(func() *VNode { return Span(Text("Hi")) })
		node := Div(comp)
		if len(node.Children) != 1 {
			t.Fatalf("Children len = %v, want 1", len(node.Children))
		}
		if node.Children[0].Kind != KindComponent {
			t.Errorf("Child kind = %v, want KindComponent", node.Children[0].Kind)
		}
	})

	t.Run("mixed attributes and children", func(t *testing.T) {
		node := Div(
			Class("card"),
			H1(Text("Title")),
			ID("main"),
			P(Text("Content")),
		)
		if node.Props["class"] != "card" {
			t.Errorf("class = %v, want card", node.Props["class"])
		}
		if node.Props["id"] != "main" {
			t.Errorf("id = %v, want main", node.Props["id"])
		}
		if len(node.Children) != 2 {
			t.Errorf("Children len = %v, want 2", len(node.Children))
		}
	})
}

func TestVoidElements(t *testing.T) {
	voids := []string{"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"}
	for _, tag := range voids {
		if !IsVoidElement(tag) {
			t.Errorf("IsVoidElement(%q) = false, want true", tag)
		}
	}

	nonVoids := []string{"div", "span", "p", "a", "button"}
	for _, tag := range nonVoids {
		if IsVoidElement(tag) {
			t.Errorf("IsVoidElement(%q) = true, want false", tag)
		}
	}
}

func TestAllElements(t *testing.T) {
	// Test a representative sample of elements
	elements := []struct {
		fn  func(...any) *VNode
		tag string
	}{
		{Div, "div"},
		{Span, "span"},
		{P, "p"},
		{H1, "h1"},
		{H2, "h2"},
		{H3, "h3"},
		{Button, "button"},
		{Input, "input"},
		{Form, "form"},
		{A, "a"},
		{Ul, "ul"},
		{Li, "li"},
		{Table, "table"},
		{Tr, "tr"},
		{Td, "td"},
		{Img, "img"},
		{Header, "header"},
		{Footer, "footer"},
		{Nav, "nav"},
		{Section, "section"},
		{Article, "article"},
	}

	for _, e := range elements {
		t.Run(e.tag, func(t *testing.T) {
			node := e.fn(Class("test"))
			if node.Kind != KindElement {
				t.Errorf("Kind = %v, want KindElement", node.Kind)
			}
			if node.Tag != e.tag {
				t.Errorf("Tag = %v, want %v", node.Tag, e.tag)
			}
		})
	}
}

func TestCustomElement(t *testing.T) {
	node := CustomElement("my-component", Class("custom"), Attr{Key: "data-value", Value: "test"})
	if node.Tag != "my-component" {
		t.Errorf("Tag = %v, want my-component", node.Tag)
	}
	if node.Props["class"] != "custom" {
		t.Errorf("class = %v, want custom", node.Props["class"])
	}
}

func TestAllElementsComprehensive(t *testing.T) {
	// Test ALL element functions to get 100% coverage on elements.go
	elements := []struct {
		fn  func(...any) *VNode
		tag string
	}{
		// Document structure
		{Html, "html"},
		{Head, "head"},
		{Body, "body"},
		{Title, "title"},
		{Meta, "meta"},
		{Link, "link"},
		{Base, "base"},

		// Content sectioning
		{Header, "header"},
		{Footer, "footer"},
		{Main, "main"},
		{Nav, "nav"},
		{Section, "section"},
		{Article, "article"},
		{Aside, "aside"},
		{Address, "address"},
		{H1, "h1"},
		{H2, "h2"},
		{H3, "h3"},
		{H4, "h4"},
		{H5, "h5"},
		{H6, "h6"},
		{Hgroup, "hgroup"},

		// Text content
		{Div, "div"},
		{P, "p"},
		{Span, "span"},
		{Pre, "pre"},
		{Blockquote, "blockquote"},
		{Ul, "ul"},
		{Ol, "ol"},
		{Li, "li"},
		{Dl, "dl"},
		{Dt, "dt"},
		{Dd, "dd"},
		{Hr, "hr"},
		{Figure, "figure"},
		{Figcaption, "figcaption"},

		// Inline text
		{A, "a"},
		{Strong, "strong"},
		{Em, "em"},
		{B, "b"},
		{I, "i"},
		{U, "u"},
		{S, "s"},
		{Small, "small"},
		{Mark, "mark"},
		{Sub, "sub"},
		{Sup, "sup"},
		{Code, "code"},
		{Kbd, "kbd"},
		{Samp, "samp"},
		{Var, "var"},
		{Abbr, "abbr"},
		{Time_, "time"},
		{Cite, "cite"},
		{Q, "q"},
		{Dfn, "dfn"},
		{Ruby, "ruby"},
		{Rt, "rt"},
		{Rp, "rp"},
		{Bdi, "bdi"},
		{Bdo, "bdo"},
		{Data, "data"},
		{Br, "br"},
		{Wbr, "wbr"},

		// Forms
		{Form, "form"},
		{Input, "input"},
		{Textarea, "textarea"},
		{Select, "select"},
		{Option, "option"},
		{Optgroup, "optgroup"},
		{Button, "button"},
		{Label, "label"},
		{Fieldset, "fieldset"},
		{Legend, "legend"},
		{Datalist, "datalist"},
		{Output, "output"},
		{Progress, "progress"},
		{Meter, "meter"},

		// Tables
		{Table, "table"},
		{Thead, "thead"},
		{Tbody, "tbody"},
		{Tfoot, "tfoot"},
		{Tr, "tr"},
		{Th, "th"},
		{Td, "td"},
		{Caption, "caption"},
		{Colgroup, "colgroup"},
		{Col, "col"},

		// Media
		{Img, "img"},
		{Picture, "picture"},
		{Source, "source"},
		{Video, "video"},
		{Audio, "audio"},
		{Track, "track"},
		{Iframe, "iframe"},
		{Embed, "embed"},
		{Object, "object"},
		{Param, "param"},
		{Canvas, "canvas"},
		{Svg, "svg"},
		{Math, "math"},
		{Map_, "map"},
		{Area, "area"},

		// Interactive
		{Details, "details"},
		{Summary, "summary"},
		{Dialog, "dialog"},
		{Menu, "menu"},

		// Scripting
		{Script, "script"},
		{Noscript, "noscript"},
		{Template, "template"},
		{Slot, "slot"},
		{Style, "style"},
	}

	for _, e := range elements {
		t.Run(e.tag, func(t *testing.T) {
			node := e.fn()
			if node.Kind != KindElement {
				t.Errorf("Kind = %v, want KindElement", node.Kind)
			}
			if node.Tag != e.tag {
				t.Errorf("Tag = %v, want %v", node.Tag, e.tag)
			}
		})
	}
}
