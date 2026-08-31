package ohm

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unicode/utf16"
)

// type TerminalVisitor[P, R any] interface {
// 	Terminal(payload P, node *TerminalNode) (result R)
// }

type TerminalVisitor interface {
	Terminal(node TerminalNode)
}

type Acceptor[P, R any] interface {
	Accept(this Node, visitor any, payload P) (result R, err error)
}

type NodeVisitor interface {
	NodeVisit(node Node)
}

type SpaceBeforeVisitor interface {
	SpaceBefore(spaces string)
}

type MakeNodeFromRoot[P, R any] func(root Node) any

// SkipCheckName is a marker interface that opts a visitor type out of the
// runtime method-signature validation performed by [TypeCheckMethod].
//
// Implement this interface on a visitor struct when you want to bypass the
// check — for example, while prototyping a visitor whose Visit* methods are
// not yet in their final form, or when a visitor intentionally deviates from
// the expected signature conventions.
//
// Usage:
//
//	var _ goohm.SkipCheckName = MyVisitor{}
//
//	func (MyVisitor) SkipCheckName() {}
type SkipCheckName interface {
	SkipCheckName()
}

func WalkCstCallTermAndSpace(
	node Node,
	fnTerm func(node TerminalNode),
	fnSpaceBefore func(spaces string),
) {
	if term, is := node.(TerminalNode); is && fnTerm != nil {
		fnTerm(term)
		return
	}
	children := node.Children()
	startIdx := node.StartIdx()
	for _, child := range children {
		// Emit any implicit spaces before this child.
		c_startIdx := child.StartIdx()
		if fnSpaceBefore != nil && c_startIdx > startIdx {
			// spacesLen := c_startIdx - startIdx
			if (c_startIdx-startIdx) > 0 && c_startIdx <= len(node.CstCtx().inputUTF16) {
				fnSpaceBefore(string(utf16.Decode(node.CstCtx().inputUTF16[startIdx:c_startIdx])))
			}
		}
		WalkCstCallTermAndSpace(child, fnTerm, fnSpaceBefore)
		startIdx = int(child.StartIdx()) + child.MatchLength()
	}
}

// Extract elements from a NonemptyListOf/EmptyListOf/ListOf CST node.
// see packages/compiler/src/buildGrammar.ts:46
func ListOfElement(node Node) []Node {
	// fmt.Printf("\n\n")
	name := node.CtorName()
	idx := strings.Index(name, "<")
	name = name[:idx]
	switch name {
	case "EmptyListOf":
		fallthrough
	case "emptyListOf":
		return []Node{}
	case "NonemptyListOf":
		fallthrough
	case "nonemptyListOf":
		first := node.Children()[0]
		restList := node.Children()[1] // (sep elem)*
		kids := make([]Node, 0, len(restList.Children())+1)
		kids = append(kids, first)
		// fmt.Printf("!%v\n", first)
		// TODO implement using i % 2 != 0
		for i, seqNode := range restList.Children() {
			ctor := seqNode.CtorName()
			if ctor == "_list" {
				panic("????")
			}
			if ctor[:1] != "_" {
				_ = i
				// fmt.Printf("* %d %v\n", i, seqNode)
				kids = append(kids, seqNode)
			}
			// fmt.Printf("!!%d %d %v\n", i, len(seqNode.Children()), seqNode)
			// if seqNode.Children() == nil || len(seqNode.Children()) == 0 {
			// 	continue
			// }
			// fmt.Printf("*%v\n", seqNode.Children()[0])
			// kids = append(kids, seqNode.Children()[0])
		}
		return kids
	case "ListOf":
		fallthrough
	case "listOf":
		// fmt.Printf("!!!%v\n", node.Children()[0])
		return ListOfElement(node.Children()[0])
	}
	panic(fmt.Sprintf("Expected ListOf node, got %s", name))
}

func AcceptBuiltinHOR[R any](node Node, fn_elem func(Node) R, fn_sep func(Node) R) {
	name := node.CtorName()
	idx := strings.Index(name, "<")
	name = name[:idx]
	switch name {
	case "EmptyListOf":
		fallthrough
	case "emptyListOf":
		return
	case "NonemptyListOf":
		fallthrough
	case "nonemptyListOf":
		first := node.Children()[0]
		fn_elem(first)
		restList := node.Children()[1] // (sep elem)*
		kids := make([]Node, 0, len(restList.Children())+1)
		kids = append(kids, first)
		// fmt.Printf("!%v\n", first)
		// TODO implement using i % 2 != 0
		for i, n := range restList.Children() {
			if i%2 == 0 {
				fn_sep(n)
			} else {
				fn_elem(n)
			}
			// ctor := n.CtorName()
			// if ctor == "_list" {
			// 	panic("????")
			// }
			// if ctor[:1] != "_" {
			// 	_ = i
			// 	// fmt.Printf("* %d %v\n", i, seqNode)
			// 	kids = append(kids, n)
			// }
			// // fmt.Printf("!!%d %d %v\n", i, len(seqNode.Children()), seqNode)
			// // if seqNode.Children() == nil || len(seqNode.Children()) == 0 {
			// // 	continue
			// // }
			// // fmt.Printf("*%v\n", seqNode.Children()[0])
			// // kids = append(kids, seqNode.Children()[0])
		}
		return
	case "ListOf":
		fallthrough
	case "listOf":
		// fmt.Printf("!!!%v\n", node.Children()[0])
		AcceptBuiltinHOR(node.Children()[0], fn_elem, fn_sep)
		return
	}
	panic(fmt.Sprintf("Expected ListOf node, got %s", name))
}

func AssertRule(node Node, rule string) {
	if node.CtorName() != rule {
		s, f := node.Source()
		panic(fmt.Errorf("excepted %s, got %s. %d-%d", rule, node.CtorName(), s, f))
	}
}

// func CheckName[T any](v any) {
// 	if _, skip := v.(SkipCheckName); skip {
// 		return
// 	}
// 	if v == nil {
// 		return
// 	}
// 	typeStr := fmt.Sprintf("%v", reflect.TypeFor[T]())
// 	dotIdx := strings.Index(typeStr, ".")
// 	osbIdx := strings.Index(typeStr, "[")
// 	csbIdx := strings.LastIndex(typeStr, "]")
// 	var (
// 		methodName string
// 	)
// 	if dotIdx > -1 && osbIdx > -1 && csbIdx > -1 {
// 		methodName = "Visit" + typeStr[dotIdx+1+len("Visitor"):osbIdx]
// 	} else {
// 		panic("unable to extract method name a signature from type. " + typeStr)
// 	}
// 	fmt.Fprintf(os.Stderr, "methodName '%s' typeStr '%s'\n", methodName, typeStr)
// 	if meth, exist := reflect.TypeOf(v).MethodByName(methodName); exist {
// 		var (
// 			signature string
// 			payload   string
// 			result    string
// 		)
// 		signature = typeStr[osbIdx+1 : csbIdx]
// 		payload, result = getTypeNames(signature)
// 		rule := methodName[len("Visit"):]
// 		received := fmt.Sprintf("%v", meth.Type)
// 		// get the stack trace and add the 4th frame to the error message to help debugging
// 		_, file, line, _ := runtime.Caller(3)
// 		see := fmt.Sprintf("%s:%d", file, line)
// 		panic(fmt.Sprintf(`%[1]s. Found method by name match, but incompatibles types.
//   expected func(<visitor>, %[3]s, %[5]s[%[3]s,%[4]s]) %[4]s
//   received %[2]s
//   For the likely call sight see:
//     %[6]s

//   Note:
//   **For advanced use-cases** with a heterogeneous set of visitor methods a cast of the node can be useful.
//   Note that the cast is generally on the parent node in the CST, and a specific Accept<specific-child> method will be called.
//   eg from the collect_vast visitor code:
//     ((*RuleDefine[any, string])(unsafe.Pointer(node))).AcceptRuleDescr(c, payload)

//	  **To skip this check**, which is not advised, the visitor can implement the SkipCheckName interface.
//	  ie implement the method SkipCheckName().
//	  `, methodName, received, payload, result, rule, see))
//		}
// }

// typeCheckKey identifies a unique TypeCheckMethod invocation. The check
// validates a static property of the visitor type, so its outcome is fully
// determined by these fields and only needs to run once per distinct key.
type typeCheckKey struct {
	visitor reflect.Type
	name    string
	payload reflect.Type
	result  reflect.Type
}

var (
	typeCheckMu    sync.RWMutex
	typeCheckCache = map[typeCheckKey]struct{}{}
)

func TypeCheckMethod[P, R any](v any, type_name string) {
	if _, skip := v.(SkipCheckName); skip {
		return
	}
	if v == nil {
		return
	}
	// Skip if this (visitor, node, P, R) combination has already been
	// verified — a struct-keyed map read is alloc-free on the hot path,
	// unlike the "Visit"+type_name concatenation and reflection below.
	key := typeCheckKey{
		visitor: reflect.TypeOf(v),
		name:    type_name,
		payload: reflect.TypeFor[P](),
		result:  reflect.TypeFor[R](),
	}
	typeCheckMu.RLock()
	_, done := typeCheckCache[key]
	typeCheckMu.RUnlock()
	if done {
		return
	}
	methodName := "Visit" + type_name
	if meth, exist := reflect.TypeOf(v).MethodByName(methodName); exist {
		// signature = typeStr[osbIdx+1 : csbIdx]
		// payload, result = getTypeNames(signature)
		payload_type := fmt.Sprintf("%v", reflect.TypeFor[P]())
		result_type := fmt.Sprintf("%v", reflect.TypeFor[R]())
		received := fmt.Sprintf("%v", meth.Type)
		// get the stack trace and add the 4th frame to the error message to help debugging
		_, file, line, _ := runtime.Caller(3)
		see := fmt.Sprintf("%s:%d", file, line)
		panic(fmt.Sprintf(`%[1]s. Found method by name match, but incompatibles types.
	  expected func(<visitor>, %[5]s[%[3]s,%[4]s], %[3]s) %[4]s
	  received %[2]s
	  For the likely call sight see:
	    %[6]s

	  Note:
	  **For advanced use-cases** with a heterogeneous set of visitor methods a cast of the node can be useful.
	  Note that the cast is generally on the parent node in the CST, and a specific Accept<specific-child> method will be called.
	  eg from the collect_vast visitor code:
	    ((*RuleDefine[any, string])(unsafe.Pointer(node))).AcceptRuleDescr(c, payload)

		  **To skip this check**, which is not advised, the visitor can implement the SkipCheckName interface.
		  ie implement the method SkipCheckName().
		  `, methodName, received, payload_type, result_type, type_name, see))
	}
	// Verified: record so subsequent visits with this key skip the work above.
	typeCheckMu.Lock()
	typeCheckCache[key] = struct{}{}
	typeCheckMu.Unlock()
}

func AssertName(this Node, name string) {
	if this.CtorName() != name {
		panic(fmt.Errorf(`name didn't name.
expected '%s'
received '%s'
`, name, this.CtorName()))
	}
}

// parse "type, type" into two strings, where type could contain generics ie "name[*,*]"
func getTypeNames(inner string) (string, string) {
	depth := 0
	for i, c := range inner {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				return strings.TrimSpace(inner[:i]), strings.TrimSpace(inner[i+1:])
			}
		}
	}
	panic("getTypeNames: could not find comma separator in: " + inner)
}
