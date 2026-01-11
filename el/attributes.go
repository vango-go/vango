// This file re-exports vdom attribute helpers for the el package.
package el

import "github.com/vango-go/vango/pkg/vdom"

func ID(id string) Attr {
	return vdom.ID(id)
}
func Class(classes ...string) Attr {
	return vdom.Class(classes...)
}
func StyleAttr(style string) Attr {
	return vdom.StyleAttr(style)
}
func Data(key, value string) Attr {
	return vdom.Data(key, value)
}
func DataAttr(key, value string) Attr {
	return vdom.DataAttr(key, value)
}
func Role(role string) Attr {
	return vdom.Role(role)
}
func AriaLabel(label string) Attr {
	return vdom.AriaLabel(label)
}
func AriaHidden(hidden bool) Attr {
	return vdom.AriaHidden(hidden)
}
func AriaExpanded(expanded bool) Attr {
	return vdom.AriaExpanded(expanded)
}
func AriaDescribedBy(id string) Attr {
	return vdom.AriaDescribedBy(id)
}
func AriaLabelledBy(id string) Attr {
	return vdom.AriaLabelledBy(id)
}
func AriaLive(mode string) Attr {
	return vdom.AriaLive(mode)
}
func AriaControls(id string) Attr {
	return vdom.AriaControls(id)
}
func AriaCurrent(value string) Attr {
	return vdom.AriaCurrent(value)
}
func AriaDisabled(disabled bool) Attr {
	return vdom.AriaDisabled(disabled)
}
func AriaPressed(pressed string) Attr {
	return vdom.AriaPressed(pressed)
}
func AriaSelected(selected bool) Attr {
	return vdom.AriaSelected(selected)
}
func AriaHasPopup(value string) Attr {
	return vdom.AriaHasPopup(value)
}
func AriaModal(modal bool) Attr {
	return vdom.AriaModal(modal)
}
func AriaAtomic(atomic bool) Attr {
	return vdom.AriaAtomic(atomic)
}
func AriaBusy(busy bool) Attr {
	return vdom.AriaBusy(busy)
}
func AriaValueNow(value float64) Attr {
	return vdom.AriaValueNow(value)
}
func AriaValueMin(value float64) Attr {
	return vdom.AriaValueMin(value)
}
func AriaValueMax(value float64) Attr {
	return vdom.AriaValueMax(value)
}
func TabIndex(index int) Attr {
	return vdom.TabIndex(index)
}
func AccessKey(key string) Attr {
	return vdom.AccessKey(key)
}
func Hidden(hidden ...bool) Attr {
	return vdom.Hidden(hidden...)
}
func TitleAttr(title string) Attr {
	return vdom.TitleAttr(title)
}
func ContentEditable(editable bool) Attr {
	return vdom.ContentEditable(editable)
}
func Draggable() Attr {
	return vdom.Draggable()
}
func Spellcheck(check bool) Attr {
	return vdom.Spellcheck(check)
}
func Lang(lang string) Attr {
	return vdom.Lang(lang)
}
func Dir(dir string) Attr {
	return vdom.Dir(dir)
}
func Href(url string) Attr {
	return vdom.Href(url)
}
func Target(target string) Attr {
	return vdom.Target(target)
}
func Rel(rel string) Attr {
	return vdom.Rel(rel)
}
func Download(filename ...string) Attr {
	return vdom.Download(filename...)
}
func Hreflang(lang string) Attr {
	return vdom.Hreflang(lang)
}
func Name(name string) Attr {
	return vdom.Name(name)
}
func Value(value string) Attr {
	return vdom.Value(value)
}
func Type(t string) Attr {
	return vdom.Type(t)
}
func Placeholder(text string) Attr {
	return vdom.Placeholder(text)
}
func Disabled(disabled ...bool) Attr {
	return vdom.Disabled(disabled...)
}
func Readonly(readonly ...bool) Attr {
	return vdom.Readonly(readonly...)
}
func Required(required ...bool) Attr {
	return vdom.Required(required...)
}
func Checked(checked ...bool) Attr {
	return vdom.Checked(checked...)
}
func Selected(selected ...bool) Attr {
	return vdom.Selected(selected...)
}
func Multiple(multiple ...bool) Attr {
	return vdom.Multiple(multiple...)
}
func Autofocus(autofocus ...bool) Attr {
	return vdom.Autofocus(autofocus...)
}
func Autocomplete(value string) Attr {
	return vdom.Autocomplete(value)
}
func Pattern(pattern string) Attr {
	return vdom.Pattern(pattern)
}
func MinLength(n int) Attr {
	return vdom.MinLength(n)
}
func MaxLength(n int) Attr {
	return vdom.MaxLength(n)
}
func Min(value string) Attr {
	return vdom.Min(value)
}
func Max(value string) Attr {
	return vdom.Max(value)
}
func Step(value string) Attr {
	return vdom.Step(value)
}
func Accept(types string) Attr {
	return vdom.Accept(types)
}
func Capture(mode string) Attr {
	return vdom.Capture(mode)
}
func Rows(n int) Attr {
	return vdom.Rows(n)
}
func Cols(n int) Attr {
	return vdom.Cols(n)
}
func Wrap(mode string) Attr {
	return vdom.Wrap(mode)
}
func Action(url string) Attr {
	return vdom.Action(url)
}
func Method(method string) Attr {
	return vdom.Method(method)
}
func Enctype(enctype string) Attr {
	return vdom.Enctype(enctype)
}
func Novalidate(novalidate ...bool) Attr {
	return vdom.Novalidate(novalidate...)
}
func For(id string) Attr {
	return vdom.For(id)
}
func FormAttr(id string) Attr {
	return vdom.FormAttr(id)
}
func Src(url string) Attr {
	return vdom.Src(url)
}
func Alt(text string) Attr {
	return vdom.Alt(text)
}
func Width(w int) Attr {
	return vdom.Width(w)
}
func Height(h int) Attr {
	return vdom.Height(h)
}
func Loading(mode string) Attr {
	return vdom.Loading(mode)
}
func Decoding(mode string) Attr {
	return vdom.Decoding(mode)
}
func Srcset(srcset string) Attr {
	return vdom.Srcset(srcset)
}
func SizesAttr(sizes string) Attr {
	return vdom.SizesAttr(sizes)
}
func Controls(controls ...bool) Attr {
	return vdom.Controls(controls...)
}
func Autoplay(autoplay ...bool) Attr {
	return vdom.Autoplay(autoplay...)
}
func Loop(loop ...bool) Attr {
	return vdom.Loop(loop...)
}
func MutedAttr(muted ...bool) Attr {
	return vdom.MutedAttr(muted...)
}
func Preload(mode string) Attr {
	return vdom.Preload(mode)
}
func Poster(url string) Attr {
	return vdom.Poster(url)
}
func Playsinline(playsinline ...bool) Attr {
	return vdom.Playsinline(playsinline...)
}
func Sandbox(value string) Attr {
	return vdom.Sandbox(value)
}
func Allow(value string) Attr {
	return vdom.Allow(value)
}
func Allowfullscreen(allowfullscreen ...bool) Attr {
	return vdom.Allowfullscreen(allowfullscreen...)
}
func Colspan(n int) Attr {
	return vdom.Colspan(n)
}
func Rowspan(n int) Attr {
	return vdom.Rowspan(n)
}
func Scope(scope string) Attr {
	return vdom.Scope(scope)
}
func HeadersAttr(ids string) Attr {
	return vdom.HeadersAttr(ids)
}
func Charset(charset string) Attr {
	return vdom.Charset(charset)
}
func Content(content string) Attr {
	return vdom.Content(content)
}
func HttpEquiv(value string) Attr {
	return vdom.HttpEquiv(value)
}
func ClassIf(condition bool, class string) Attr {
	return vdom.ClassIf(condition, class)
}
func AttrIf(condition bool, a Attr) Attr {
	return vdom.AttrIf(condition, a)
}
func Classes(classes ...any) Attr {
	return vdom.Classes(classes...)
}
func Open(open ...bool) Attr {
	return vdom.Open(open...)
}
func Defer_(deferAttr ...bool) Attr {
	return vdom.Defer_(deferAttr...)
}
func Async(async ...bool) Attr {
	return vdom.Async(async...)
}
func Crossorigin(value string) Attr {
	return vdom.Crossorigin(value)
}
func Integrity(value string) Attr {
	return vdom.Integrity(value)
}
func List(id string) Attr {
	return vdom.List(id)
}
func Inputmode(mode string) Attr {
	return vdom.Inputmode(mode)
}
func Enterkeyhint(hint string) Attr {
	return vdom.Enterkeyhint(hint)
}
