package urlparam

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
)

func TestInitialURLState_ConsumeOnce(t *testing.T) {
	state := &InitialURLState{
		Path:   "/projects",
		Params: map[string]string{"page": "2"},
	}
	if state.IsConsumed() {
		t.Fatalf("IsConsumed: got true, want false")
	}
	if got := state.Consume(); got == nil || got["page"] != "2" {
		t.Fatalf("Consume: got %v, want params", got)
	}
	if !state.IsConsumed() {
		t.Fatalf("IsConsumed: got false, want true")
	}
	if got := state.Consume(); got != nil {
		t.Fatalf("Consume second time: got %v, want nil", got)
	}
}

func TestNavigator_NavigateQueuesCorrectPatch(t *testing.T) {
	var got []protocol.Patch
	nav := NewNavigator(func(p protocol.Patch) {
		got = append(got, p)
	})

	params := map[string]string{"q": "go"}

	nav.Navigate(params, ModePush)
	if len(got) != 1 || got[0].Op != protocol.PatchURLPush || got[0].Params["q"] != "go" {
		t.Fatalf("ModePush patch: got %#v", got)
	}

	nav.Navigate(params, ModeReplace)
	if len(got) != 2 || got[1].Op != protocol.PatchURLReplace {
		t.Fatalf("ModeReplace patch: got %#v", got)
	}

	nav.Navigate(params, URLMode(123)) // default branch should push
	if len(got) != 3 || got[2].Op != protocol.PatchURLPush {
		t.Fatalf("unknown mode patch: got %#v", got)
	}

	NewNavigator(nil).Navigate(params, ModePush) // should be a no-op, no panic
}

func TestURLParam_SetAndUpdate_NavigateModeAndParams(t *testing.T) {
	p := Param("page", 1, Replace)

	type navCall struct {
		params map[string]string
		mode   URLMode
	}
	calls := make(chan navCall, 10)
	p.SetNavigator(func(params map[string]string, mode URLMode) {
		calls <- navCall{params: params, mode: mode}
	})

	p.Set(2)
	select {
	case c := <-calls:
		if c.mode != ModeReplace || c.params["page"] != "2" {
			t.Fatalf("Set navigate: got mode=%v params=%v", c.mode, c.params)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Set navigate: timed out waiting for call")
	}

	p.Update(func(v int) int { return v + 1 })
	select {
	case c := <-calls:
		if c.mode != ModeReplace || c.params["page"] != "3" {
			t.Fatalf("Update navigate: got mode=%v params=%v", c.mode, c.params)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Update navigate: timed out waiting for call")
	}
}

func TestURLParam_SetFromURL_DoesNotNavigate(t *testing.T) {
	p := Param("q", "default", Replace)
	navigateCalls := make(chan struct{}, 1)
	p.SetNavigator(func(params map[string]string, mode URLMode) {
		navigateCalls <- struct{}{}
	})

	if err := p.SetFromURL(map[string]string{"q": "from-url"}); err != nil {
		t.Fatalf("SetFromURL: %v", err)
	}
	select {
	case <-navigateCalls:
		t.Fatal("SetFromURL should not call navigator")
	case <-time.After(40 * time.Millisecond):
		// ok
	}
}

func TestURLParam_Debounce_CoalescesURLUpdates(t *testing.T) {
	p := Param("q", "", Replace, Debounce(25*time.Millisecond))
	got := make(chan string, 10)
	p.SetNavigator(func(params map[string]string, mode URLMode) {
		got <- params["q"]
	})

	p.Set("a")
	p.Set("ab")

	select {
	case v := <-got:
		t.Fatalf("navigate called too early: %q", v)
	default:
	}

	select {
	case v := <-got:
		if v != "ab" {
			t.Fatalf("debounced value: got %q, want %q", v, "ab")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for debounced navigate call")
	}

	select {
	case v := <-got:
		t.Fatalf("expected single debounced navigate call, got extra %q", v)
	default:
	}
}

func TestURLParam_SerializeFlat_StructTagsSkipZero(t *testing.T) {
	type Filters struct {
		Category string `url:"cat"`
		SortBy   string `url:"sort"`
		Page     int    `url:"page"`
		Hidden   string `url:"-"`
		Untagged bool
	}

	u := &URLParam[Filters]{key: "", config: urlParamConfig{encoding: EncodingFlat}}

	params := u.serialize(Filters{
		Category: "electronics",
		Page:     2,
		Untagged: true,
	})

	if params["cat"] != "electronics" || params["page"] != "2" || params["untagged"] != "true" {
		t.Fatalf("flat serialize: got %v", params)
	}
	if _, ok := params["sort"]; ok {
		t.Fatalf("flat serialize should skip zero value field 'sort': got %v", params)
	}
	if _, ok := params["hidden"]; ok {
		t.Fatalf("flat serialize should skip '-' tagged field: got %v", params)
	}
}

func TestURLParam_Serialize_EncodingFallbackForWrongKind(t *testing.T) {
	uFlat := &URLParam[int]{key: "page", config: urlParamConfig{encoding: EncodingFlat}}
	if got := uFlat.serialize(7)["page"]; got != "7" {
		t.Fatalf("flat serialize non-struct: got %q, want %q", got, "7")
	}

	uComma := &URLParam[int]{key: "n", config: urlParamConfig{encoding: EncodingComma}}
	if got := uComma.serialize(7)["n"]; got != "7" {
		t.Fatalf("comma serialize non-slice: got %q, want %q", got, "7")
	}
}

func TestURLParam_SerializeComma_Slice(t *testing.T) {
	u := &URLParam[[]int]{key: "nums", config: urlParamConfig{encoding: EncodingComma}}
	if got := u.serialize([]int{1, 2, 3})["nums"]; got != "1,2,3" {
		t.Fatalf("comma serialize: got %q, want %q", got, "1,2,3")
	}
}

func TestURLParam_SerializeJSON_RoundTripAndMarshalError(t *testing.T) {
	type Filter struct {
		Category string `json:"cat"`
		MaxPrice int    `json:"max_price"`
	}

	u := &URLParam[Filter]{key: "filter", config: urlParamConfig{encoding: EncodingJSON}}
	encoded := u.serialize(Filter{Category: "electronics", MaxPrice: 1000})["filter"]
	if encoded == "" {
		t.Fatal("expected non-empty JSON encoding")
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}

	var roundTripped Filter
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}
	if roundTripped.Category != "electronics" || roundTripped.MaxPrice != 1000 {
		t.Fatalf("round-trip: got %+v", roundTripped)
	}

	type Bad struct {
		C chan int `json:"c"`
	}
	uBad := &URLParam[Bad]{key: "bad", config: urlParamConfig{encoding: EncodingJSON}}
	if got := uBad.serialize(Bad{C: make(chan int)})["bad"]; got != "" {
		t.Fatalf("marshal error should yield empty string: got %q", got)
	}
}

func TestURLParam_DeserializeJSON_Errors(t *testing.T) {
	u := &URLParam[map[string]any]{key: "v", config: urlParamConfig{encoding: EncodingJSON}}

	if _, err := u.deserialize(map[string]string{"v": "!!!not-base64!!!"}); err == nil {
		t.Fatal("expected base64 decode error")
	}

	encodedNotJSON := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	if _, err := u.deserialize(map[string]string{"v": encodedNotJSON}); err == nil {
		t.Fatal("expected JSON unmarshal error")
	}
}

func TestURLParam_DeserializeComma_ErrorsAndParse(t *testing.T) {
	uGood := &URLParam[[]float64]{key: "vals", config: urlParamConfig{encoding: EncodingComma}}
	got, err := uGood.deserialize(map[string]string{"vals": "1.5,2,3.25"})
	if err != nil {
		t.Fatalf("deserialize float slice: %v", err)
	}
	if !reflect.DeepEqual(got, []float64{1.5, 2, 3.25}) {
		t.Fatalf("deserialize float slice: got %v", got)
	}

	uWrongKind := &URLParam[string]{key: "v", config: urlParamConfig{encoding: EncodingComma}}
	if _, err := uWrongKind.deserialize(map[string]string{"v": "a,b"}); err == nil {
		t.Fatal("expected error for non-slice type with comma encoding")
	}

	uBadElem := &URLParam[[]int]{key: "nums", config: urlParamConfig{encoding: EncodingComma}}
	if _, err := uBadElem.deserialize(map[string]string{"nums": "1,x,3"}); err == nil {
		t.Fatal("expected element parse error")
	}
}

func TestURLParam_DeserializeFlat_ErrorsAndNonStruct(t *testing.T) {
	uNonStruct := &URLParam[uint]{key: "u", config: urlParamConfig{encoding: EncodingFlat}}
	got, err := uNonStruct.deserialize(map[string]string{"u": "42"})
	if err != nil || got != 42 {
		t.Fatalf("flat deserialize non-struct: got=%v err=%v", got, err)
	}

	type Bad struct {
		X struct{} `url:"x"`
	}
	uBad := &URLParam[Bad]{key: "", config: urlParamConfig{encoding: EncodingFlat}}
	if _, err := uBad.deserialize(map[string]string{"x": "nope"}); err == nil {
		t.Fatal("expected error for unsupported struct field type")
	}

	type WithUnexported struct {
		Public  int `url:"p"`
		private int `url:"priv"`
	}
	uIgnore := &URLParam[WithUnexported]{key: "", config: urlParamConfig{encoding: EncodingFlat}}
	parsed, err := uIgnore.deserialize(map[string]string{"p": "7", "priv": "9"})
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if parsed.Public != 7 {
		t.Fatalf("public field: got %v, want 7", parsed.Public)
	}
	if parsed.private != 0 {
		t.Fatalf("unexported field should not be set: got %v, want 0", parsed.private)
	}
}

func TestURLParam_ParseValue_UnsupportedType(t *testing.T) {
	u := &URLParam[struct{}]{key: "x"}
	if _, err := u.parseValue("anything"); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestURLParam_HookSlotStableIdentityAndInitialHydration(t *testing.T) {
	owner := vango.NewOwner(nil)
	initial := &InitialURLState{Params: map[string]string{"page": "5"}}

	var p1 *URLParam[int]
	vango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()

		vango.SetContext(InitialParamsKey, initial)
		p1 = Param("page", 1)
	})

	if p1.Get() != 5 {
		t.Fatalf("initial hydration: got %v, want 5", p1.Get())
	}
	if !initial.IsConsumed() {
		t.Fatalf("expected initial state to be consumed after first URLParam")
	}

	p1.Set(6)

	var p2 *URLParam[int]
	vango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()

		p2 = Param("page", 1)
	})

	if p2 != p1 {
		t.Fatalf("stable identity: got %p, want %p", p2, p1)
	}
	if p2.Get() != 6 {
		t.Fatalf("value preserved across renders: got %v, want 6", p2.Get())
	}
}

