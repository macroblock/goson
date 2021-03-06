package main //package gocc

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type (
	itemType int

	item struct {
		typ itemType
		val string
	}
)

const (
	itemError itemType = iota // error
	itemEOF
	itemText // any text outside meta tags
	itemLeftMeta
	itemRightMeta
	itemIdentifier
)

const (
	leftMeta  = "{{"
	rightMeta = "}}"
	runeEOF   = 0
)

func (i item) String() string {
	prefix := ""
	switch i.typ {
	case itemEOF:
		return "t: EOF"
	case itemError:
		return "t: Error; v: " + i.val
	case itemText:
		prefix = "t: Text; "
	case itemLeftMeta:
		prefix = "t: LeftMeta; "
	case itemRightMeta:
		prefix = "t: RightMeta; "
	case itemIdentifier:
		prefix = "t: Identifier; "
	}
	if len(i.val) > 10 {
		return fmt.Sprintf("%s v: %.10q...", prefix, i.val)
	}
	return fmt.Sprintf("%s v: %q", prefix, i.val)
}

// lexer
type lexer struct {
	name  string // error name
	src   string
	start int
	pos   int
	width int
	items chan item
}

type stateFn func(*lexer) stateFn

//func lex(name, src string) *lexer {
//	l := &lexer{
//		name: name,
//		src:  src,
//		//state: lexOutsideAction,
//		items: make(chan item, 2),
//	}
//	return l
//}

func lex(name, src string) (*lexer, chan item) {
	l := &lexer{
		name:  name,
		src:   src,
		items: make(chan item),
	}
	go l.run()
	return l, l.items
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

// emit
func (l *lexer) emit(t itemType) {
	//fmt.Println(item{t, l.src[l.start:l.pos]})
	l.items <- item{t, l.src[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() rune {
	if l.pos >= len(l.src) {
		l.width = 0
		return runeEOF
	}
	r, w := utf8.DecodeRuneInString(l.src[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	ch := l.next()
	l.backup()
	return ch
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) run() {
	for state := lexOutsideAction; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
	panic("not reached: nextItem")
}

func lexOutsideAction(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.src[l.pos:], leftMeta) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLeftMeta
		}
		if l.next() == runeEOF {
			break
		}
	}
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(itemEOF)
	return nil
}

func lexLeftMeta(l *lexer) stateFn {
	l.pos += len(leftMeta)
	l.emit(itemLeftMeta)
	return lexInsideAction
}

func lexRightMeta(l *lexer) stateFn {
	l.pos += len(rightMeta)
	l.emit(itemRightMeta)
	return lexOutsideAction
}

func lexIdentifier(l *lexer) stateFn {
	if unicode.IsLetter(l.next()) {
		for r := l.next(); unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'; r = l.next() {
		}
	} else {
		panic("not reached: lexIdentifier")
	}
	l.backup()
	l.emit(itemIdentifier)
	return lexInsideAction
}

func lexInsideAction(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.src[l.pos:], rightMeta) {
			return lexRightMeta
		}
		switch r := l.next(); {
		case r == runeEOF || r == '\n':
			return l.errorf("unclosed action")
		case unicode.IsSpace(r):
			l.ignore()
		case unicode.IsLetter(r):
			l.backup()
			return lexIdentifier
		default:
			return l.errorf("unexpected symbol %q", r)
		}
	}
}

func main() {
	fmt.Println("!!! begin.")
	_, ch := lex("oops", "test a b c {{d e f 1 some_Text}} xxx")
	for item := range ch {
		if ch == nil {
			break
		}
		fmt.Println("!!! ", item)
	}
	fmt.Println("!!! end.")
}
