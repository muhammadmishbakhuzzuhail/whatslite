// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// richtext_emoji.go — FORK minimal dari gioui.org/x/styledtext + x/richtext
// (Unlicense) dgn SATU perbaikan penting: paintGlyph juga menggambar
// `shaper.Bitmaps(line)` (glyph bitmap/warna spt NotoColorEmoji). Versi upstream
// HANYA menggambar outline vektor (`clip.Outline{Path: shaper.Shape(line)}`) → emoji
// warna (CBDT) tak punya outline → KOSONG. gio core widget.Label sudah menggambar
// keduanya; styledtext tidak. Tanpa fork ini, teks ber-emoji di bubble (yg pakai
// rich-text utk *format*/URL/@mention) jadi blank di posisi emoji.
//
// API publik di-prefiks `rt` (rtSpanStyle/rtInteractiveText/rtText/rtClick) agar
// tak bentrok dgn paket; tipe styledtext internal di-prefiks `st`.
package gioui

import (
	"image"
	"image/color"
	"time"
	"unicode/utf8"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"
)

// ---- styledtext (internal) ----

type stSpanStyle struct {
	Font    font.Font
	Size    unit.Sp
	Color   color.NRGBA
	Content string
	idx     int
}

func (ss stSpanStyle) layout(gtx layout.Context, shape stSpanShape) layout.Dimensions {
	paint.ColorOp{Color: ss.Color}.Add(gtx.Ops)
	defer op.Offset(shape.offset).Push(gtx.Ops).Pop()
	shape.call.Add(gtx.Ops)
	return layout.Dimensions{Size: shape.size}
}

type stSpanShape struct {
	offset image.Point
	size   image.Point
	call   op.CallOp
	ascent int
}

type stSpanResults struct {
	call             op.CallOp
	width            int
	height           int
	ascent           int
	runes            int
	multiLine        bool
	endedWithNewline bool
}

type stTextStyle struct {
	Styles          []stSpanStyle
	LineHeight      unit.Sp
	LineHeightScale float32
	*text.Shaper
}

func stText(shaper *text.Shaper, styles ...stSpanStyle) stTextStyle {
	return stTextStyle{Styles: styles, Shaper: shaper}
}

func (t stTextStyle) iterateSpan(gtx layout.Context, maxWidth int, span stSpanStyle, truncate bool) (op.CallOp, stTextIterator) {
	var glyphs [32]text.Glyph
	maxLines := 0
	if truncate {
		maxLines = 1
	}
	lineHeight := fixed.I(gtx.Sp(t.LineHeight))
	macro := op.Record(gtx.Ops)
	paint.ColorOp{Color: span.Color}.Add(gtx.Ops)
	t.Shaper.LayoutString(text.Parameters{
		Font:            span.Font,
		PxPerEm:         fixed.I(gtx.Sp(span.Size)),
		MaxLines:        maxLines,
		MaxWidth:        maxWidth,
		Truncator:       "​",
		Locale:          gtx.Locale,
		WrapPolicy:      text.WrapWords,
		LineHeight:      lineHeight,
		LineHeightScale: t.LineHeightScale,
	}, span.Content)
	ti := stTextIterator{
		viewport: image.Rectangle{Max: gtx.Constraints.Max},
		maxLines: 1,
	}
	line := glyphs[:0]
	for g, ok := t.Shaper.NextGlyph(); ok; g, ok = t.Shaper.NextGlyph() {
		line, ok = ti.paintGlyph(gtx, t.Shaper, g, line)
		if !ok {
			break
		}
	}
	return macro.Stop(), ti
}

func (t stTextStyle) layoutSpan(gtx layout.Context, maxWidth int, span stSpanStyle) stSpanResults {
	call, ti := t.iterateSpan(gtx, maxWidth, span, true)
	runesDisplayed := ti.runes
	multiLine := runesDisplayed < utf8.RuneCountInString(span.Content)
	endedWithNewline := ti.hasNewline
	if multiLine {
		var i int
		for i = 0; i < runesDisplayed; {
			_, sz := utf8.DecodeRuneInString(span.Content[i:])
			i += sz
		}
		firstTruncatedRune, _ := utf8.DecodeRuneInString(span.Content[i:])
		if firstTruncatedRune == '\n' {
			endedWithNewline = true
			runesDisplayed++
		} else if runesDisplayed == 0 {
			call, ti = t.iterateSpan(gtx, maxWidth, span, false)
			runesDisplayed = ti.runes
			multiLine = runesDisplayed < utf8.RuneCountInString(span.Content)
			endedWithNewline = ti.hasNewline
		}
	}
	return stSpanResults{
		call:             call,
		width:            ti.bounds.Dx(),
		height:           ti.bounds.Dy(),
		ascent:           ti.baseline,
		runes:            runesDisplayed,
		multiLine:        multiLine,
		endedWithNewline: endedWithNewline,
	}
}

func (t stTextStyle) Layout(gtx layout.Context, spanFn func(gtx layout.Context, idx int, dims layout.Dimensions)) layout.Dimensions {
	spans := make([]stSpanStyle, len(t.Styles))
	copy(spans, t.Styles)
	for i := range spans {
		spans[i].idx = i
	}
	lineHeightScale := t.LineHeightScale
	lineHeightPx := gtx.Sp(t.LineHeight)
	if lineHeightScale == 0 {
		lineHeightScale = 1.2
	}
	var (
		lineDims       image.Point
		lineAscent     int
		overallSize    image.Point
		lineShapes     []stSpanShape
		lineStartIndex int
	)
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		maxWidth := gtx.Constraints.Max.X - lineDims.X
		res := t.layoutSpan(gtx, maxWidth, span)
		forceToNextLine := lineDims.X > 0 && res.width > maxWidth
		if !forceToNextLine {
			lineShapes = append(lineShapes, stSpanShape{
				offset: image.Point{X: lineDims.X},
				size:   image.Point{X: res.width, Y: res.height},
				call:   res.call,
				ascent: res.ascent,
			})
			lineDims.X += res.width
			if lineDims.Y < res.height {
				lineDims.Y = res.height
			}
			if lineAscent < res.ascent {
				lineAscent = res.ascent
			}
			if overallSize.X < lineDims.X {
				overallSize.X = lineDims.X
			}
		}
		if res.multiLine || res.endedWithNewline || i == len(spans)-1 || forceToNextLine {
			lineMacro := op.Record(gtx.Ops)
			for j, shape := range lineShapes {
				span = spans[j+lineStartIndex]
				shape.offset.Y = overallSize.Y
				span.layout(gtx, shape)
				if spanFn == nil {
					continue
				}
				offStack := op.Offset(shape.offset).Push(gtx.Ops)
				fnGtx := gtx
				fnGtx.Constraints.Min = image.Point{}
				fnGtx.Constraints.Max = shape.size
				spanFn(fnGtx, span.idx, layout.Dimensions{Size: shape.size, Baseline: shape.ascent})
				offStack.Pop()
			}
			lineCall := lineMacro.Stop()
			lineCall.Add(gtx.Ops)
			lineShapes = lineShapes[:0]
			effectiveLineHeight := lineDims.Y
			if t.LineHeight != 0 {
				effectiveLineHeight = lineHeightPx
			}
			effectiveLineHeight = int(float32(effectiveLineHeight) * lineHeightScale)
			overallSize.Y += effectiveLineHeight
			lineDims = image.Point{}
			lineAscent = 0
		}
		if res.multiLine && !forceToNextLine {
			lineStartIndex = i + 1
			spans = append(spans, stSpanStyle{})
			for k := len(spans) - 1; k > i+1; k-- {
				spans[k] = spans[k-1]
			}
			byteLen := 0
			for r := 0; r < res.runes; r++ {
				_, n := utf8.DecodeRuneInString(span.Content[byteLen:])
				byteLen += n
			}
			span.Content = span.Content[byteLen:]
			spans[i+1] = span
		} else if forceToNextLine {
			lineStartIndex = i
			i--
		} else if res.endedWithNewline {
			lineStartIndex = i + 1
		}
	}
	return layout.Dimensions{Size: gtx.Constraints.Constrain(overallSize)}
}

// ---- glyph iterator (with emoji-bitmap fix) ----

type stTextIterator struct {
	viewport   image.Rectangle
	maxLines   int
	linesSeen  int
	init       bool
	firstX     fixed.Int26_6
	hasNewline bool
	lineOff    image.Point
	padding    image.Rectangle
	bounds     image.Rectangle
	runes      int
	visible    bool
	first      bool
	baseline   int
}

func (it *stTextIterator) processGlyph(g text.Glyph, ok bool) (text.Glyph, bool) {
	logicalBounds := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}
	if g.Flags&text.FlagTruncator != 0 {
		if it.runes == 0 {
			it.hasNewline = true
		}
		it.bounds.Min.Y = min(it.bounds.Min.Y, logicalBounds.Min.Y)
		it.bounds.Max.Y = max(it.bounds.Max.Y, logicalBounds.Max.Y)
		return g, false
	}
	it.runes += int(g.Runes)
	it.hasNewline = it.hasNewline || (g.Flags&text.FlagLineBreak > 0 && g.Flags&text.FlagParagraphBreak > 0)
	if it.maxLines > 0 {
		if g.Flags&text.FlagLineBreak != 0 {
			it.linesSeen++
		}
		if it.linesSeen == it.maxLines && g.Flags&text.FlagParagraphBreak != 0 {
			return g, false
		}
	}
	if d := g.Bounds.Min.X.Floor(); d < it.padding.Min.X {
		it.padding.Min.X = d
	}
	if d := (g.Bounds.Max.X - g.Advance).Ceil(); d > it.padding.Max.X {
		it.padding.Max.X = d
	}
	if !it.first {
		it.first = true
		it.baseline = int(g.Y)
		it.bounds = logicalBounds
	}
	above := logicalBounds.Max.Y < it.viewport.Min.Y
	below := logicalBounds.Min.Y > it.viewport.Max.Y
	left := logicalBounds.Max.X < it.viewport.Min.X
	right := logicalBounds.Min.X > it.viewport.Max.X
	it.visible = !above && !below && !left && !right
	if it.visible {
		it.bounds.Min.X = min(it.bounds.Min.X, logicalBounds.Min.X)
		it.bounds.Min.Y = min(it.bounds.Min.Y, logicalBounds.Min.Y)
		it.bounds.Max.X = max(it.bounds.Max.X, logicalBounds.Max.X)
		it.bounds.Max.Y = max(it.bounds.Max.Y, logicalBounds.Max.Y)
	}
	return g, ok && !below
}

func (it *stTextIterator) paintGlyph(gtx layout.Context, shaper *text.Shaper, glyph text.Glyph, line []text.Glyph) ([]text.Glyph, bool) {
	_, visibleOrBefore := it.processGlyph(glyph, true)
	if it.visible {
		if !it.init {
			it.firstX = glyph.X
			it.init = true
		}
		if len(line) == 0 {
			it.lineOff = image.Point{X: (glyph.X - it.firstX).Floor(), Y: int(glyph.Y)}.Sub(it.viewport.Min)
		}
		line = append(line, glyph)
	}
	if glyph.Flags&text.FlagLineBreak > 0 || cap(line)-len(line) == 0 || !visibleOrBefore {
		t := op.Offset(it.lineOff).Push(gtx.Ops)
		outline := clip.Outline{Path: shaper.Shape(line)}.Op().Push(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		// FIX vs upstream: gambar juga glyph bitmap/warna (emoji NotoColorEmoji).
		if call := shaper.Bitmaps(line); call != (op.CallOp{}) {
			call.Add(gtx.Ops)
		}
		t.Pop()
		line = line[:0]
	}
	return line, visibleOrBefore
}

// ---- richtext (interactive wrapper) ----

var rtLongPressDuration = 250 * time.Millisecond

type rtEventType uint8

const (
	rtHover rtEventType = iota
	rtUnhover
	rtLongPress
	rtClick
)

type rtEvent struct {
	Type      rtEventType
	ClickData gesture.ClickEvent
}

type rtInteractiveSpan struct {
	click        gesture.Click
	pressing     bool
	hovering     bool
	longPressed  bool
	pressStarted time.Time
	contents     string
	metadata     map[string]interface{}
}

func (i *rtInteractiveSpan) Update(gtx layout.Context) (rtEvent, bool) {
	if i == nil {
		return rtEvent{}, false
	}
	for {
		e, ok := i.click.Update(gtx.Source)
		if !ok {
			break
		}
		switch e.Kind {
		case gesture.KindClick:
			i.pressing = false
			if i.longPressed {
				i.longPressed = false
			} else {
				return rtEvent{Type: rtClick, ClickData: e}, true
			}
		case gesture.KindPress:
			i.pressStarted = gtx.Now
			i.pressing = true
		case gesture.KindCancel:
			i.pressing = false
			i.longPressed = false
		}
	}
	if isHovered := i.click.Hovered(); isHovered != i.hovering {
		i.hovering = isHovered
		if isHovered {
			return rtEvent{Type: rtHover}, true
		}
		return rtEvent{Type: rtUnhover}, true
	}
	if !i.longPressed && i.pressing && gtx.Now.Sub(i.pressStarted) > rtLongPressDuration {
		i.longPressed = true
		return rtEvent{Type: rtLongPress}, true
	}
	return rtEvent{}, false
}

func (i *rtInteractiveSpan) Layout(gtx layout.Context) layout.Dimensions {
	for {
		_, ok := i.Update(gtx)
		if !ok {
			break
		}
	}
	if i.pressing && !i.longPressed {
		gtx.Execute(op.InvalidateCmd{})
	}
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	pointer.CursorPointer.Add(gtx.Ops)
	i.click.Add(gtx.Ops)
	return layout.Dimensions{}
}

// Get melihat metadata pada span interaktif.
func (i *rtInteractiveSpan) Get(key string) interface{} { return i.metadata[key] }

type rtInteractiveText struct {
	Spans       []rtInteractiveSpan
	lastUpdate  time.Time
	updateIndex int
}

func (i *rtInteractiveText) resize(n int) {
	if n == 0 && i == nil {
		return
	}
	if cap(i.Spans) >= n {
		i.Spans = i.Spans[:n]
	} else {
		i.Spans = make([]rtInteractiveSpan, n)
	}
}

func (i *rtInteractiveText) Update(gtx layout.Context) (*rtInteractiveSpan, rtEvent, bool) {
	if i == nil {
		return nil, rtEvent{}, false
	}
	if i.lastUpdate != gtx.Now {
		i.lastUpdate = gtx.Now
		i.updateIndex = 0
	}
	for k := i.updateIndex; k < len(i.Spans); k++ {
		i.updateIndex = k
		span := &i.Spans[k]
		for {
			ev, ok := span.Update(gtx)
			if !ok {
				break
			}
			return span, ev, true
		}
	}
	return nil, rtEvent{}, false
}

type rtSpanStyle struct {
	Font           font.Font
	Size           unit.Sp
	Color          color.NRGBA
	Content        string
	Interactive    bool
	metadata       map[string]interface{}
	interactiveIdx int
}

// Set menetapkan metadata key→value pada span (mis. "url"). value "" → hapus.
func (ss *rtSpanStyle) Set(key string, value interface{}) {
	if value == "" {
		if ss.metadata != nil {
			delete(ss.metadata, key)
			if len(ss.metadata) == 0 {
				ss.metadata = nil
			}
		}
		return
	}
	if ss.metadata == nil {
		ss.metadata = make(map[string]interface{})
	}
	ss.metadata[key] = value
}

type rtTextStyle struct {
	State  *rtInteractiveText
	Styles []rtSpanStyle
	*text.Shaper
}

func rtText(state *rtInteractiveText, shaper *text.Shaper, styles ...rtSpanStyle) rtTextStyle {
	return rtTextStyle{State: state, Styles: styles, Shaper: shaper}
}

func (t rtTextStyle) Layout(gtx layout.Context) layout.Dimensions {
	for {
		_, _, ok := t.State.Update(gtx)
		if !ok {
			break
		}
	}
	styles := make([]stSpanStyle, len(t.Styles))
	numInteractive := 0
	for i := range t.Styles {
		st := &t.Styles[i]
		if st.Interactive {
			st.interactiveIdx = numInteractive
			numInteractive++
		}
		styles[i] = stSpanStyle{Font: st.Font, Size: st.Size, Color: st.Color, Content: st.Content}
	}
	t.State.resize(numInteractive)
	txt := stText(t.Shaper, styles...)
	return txt.Layout(gtx, func(gtx layout.Context, i int, _ layout.Dimensions) {
		span := &t.Styles[i]
		if !span.Interactive {
			return
		}
		state := &t.State.Spans[span.interactiveIdx]
		state.contents = span.Content
		state.metadata = span.metadata
		state.Layout(gtx)
	})
}
