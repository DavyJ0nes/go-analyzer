package twofer

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/exercism/go-analyzer/suggester/sugg"
	"github.com/tehsphinx/astrav"
)

// Register registers all suggestion functions for this exercise.
var Register = sugg.Register{
	Funcs: []sugg.SuggestionFunc{
		examMainFunc,
		examPlusUsed,
		examGeneralizeNames,
		examFmt,
		examComments,
		examConditional,
		examStringsJoin,
		examStringsBuilder,
		examExtraVariable,
		examExtraFunction,
		examStringsTrimSpace,
	},
	Severity: severity,
}

func examMainFunc(pkg *astrav.Package, suggs sugg.Suggester) {
	main := pkg.FuncDeclByName("ShareWith")
	if main == nil {
		suggs.AppendUnique(MissingMainFunc)
		return
	}

	if len(main.Params().Children()) != 1 {
		suggs.AppendUnique(FuncSignatureChanged)
	}
	if len(main.Results().Children()) != 1 {
		suggs.AppendUnique(FuncSignatureChanged)
	}
}

func examStringsTrimSpace(pkg *astrav.Package, suggs sugg.Suggester) {
	nodes := pkg.FindByName("TrimSpace")
	if len(nodes) != 0 {
		suggs.AppendUnique(TrimSpace)
	}
}

func examExtraFunction(pkg *astrav.Package, suggs sugg.Suggester) {
	nodes := pkg.FindByNodeType(astrav.NodeTypeFuncDecl)
	main := pkg.FuncDeclByName("main")
	if 1 < len(nodes) && main == nil {
		suggs.AppendUnique(ExtraFunction)
	}
}

func examExtraVariable(pkg *astrav.Package, suggs sugg.Suggester) {
	main := pkg.FuncDeclByName("ShareWith")
	if main == nil {
		return
	}

	params := main.Params().Children()
	if len(params) != 1 {
		suggs.AppendUnique(FuncSignatureChanged)
		return
	}
	paramName := params[0].(astrav.Named).NodeName()

	decls := main.FindByNodeType(astrav.NodeTypeAssignStmt)
	for _, decl := range decls {
		right := decl.(*astrav.AssignStmt).RHS()

		for _, node := range right {
			if !node.IsNodeType(astrav.NodeTypeIdent) {
				continue
			}
			if node.(astrav.Named).NodeName().Name == paramName.Name {
				suggs.AppendUnique(ExtraNameVar)
			}
		}

		left := decl.(*astrav.AssignStmt).LHS()
		for _, node := range left {
			if !node.IsNodeType(astrav.NodeTypeIdent) {
				continue
			}
			varName := node.(astrav.Named).NodeName().Name
			if varName != paramName.Name {
				suggs.AppendUniquePH(ExtraVar, map[string]string{
					"name": varName,
				})
			}

		}
	}
}

func examStringsJoin(pkg *astrav.Package, suggs sugg.Suggester) {
	node := pkg.FindFirstByName("Join")
	if node != nil {
		suggs.AppendUnique(StringsJoin)
	}
}

func examStringsBuilder(pkg *astrav.Package, suggs sugg.Suggester) {
	node := pkg.FindFirstByName("Builder")
	if node != nil {
		suggs.AppendUnique(StringsBuilder)
	}
}

func examPlusUsed(pkg *astrav.Package, suggs sugg.Suggester) {
	main := pkg.FuncDeclByName("ShareWith")
	if main == nil {
		suggs.AppendUnique(MissingMainFunc)
		return
	}
	nodes := main.FindByNodeType(astrav.NodeTypeBinaryExpr)

	var plusUsed bool
	for _, node := range nodes {
		expr, ok := node.(*astrav.BinaryExpr)
		if !ok {
			continue
		}
		if expr.Op.String() == "+" {
			plusUsed = true
		}
	}
	if plusUsed {
		suggs.AppendUnique(PlusUsed)
	}
}

func examFmt(pkg *astrav.Package, suggs sugg.Suggester) {
	nodes := pkg.FindByName("Sprintf")

	var spfCount int
	for _, fmtSprintf := range nodes {
		if !fmtSprintf.IsNodeType(astrav.NodeTypeSelectorExpr) {
			continue
		}

		spfCount++
		if 1 < spfCount {
			suggs.AppendUnique(MinimalConditional)
		}
	}

	nodes = pkg.FindByNodeType(astrav.NodeTypeBasicLit)
	for _, node := range nodes {
		bLit := node.(*astrav.BasicLit)
		if bytes.Contains(bLit.GetSource(), []byte("%v")) {
			suggs.AppendUnique(UseStringPH)
		}
	}
}

func examComments(pkg *astrav.Package, suggs sugg.Suggester) {
	if bytes.Contains(pkg.GetSource(), []byte("stub")) {
		suggs.AppendUnique(StubComments)
	}

	// TODO: what if there are multiple files??
	file := pkg.ChildByNodeType(astrav.NodeTypeFile)
	if file != nil {
		cGroup := file.ChildByNodeType(astrav.NodeTypeCommentGroup)
		checkComment(cGroup, suggs, "package", "twofer")
	}

	main := pkg.FuncDeclByName("ShareWith")
	if main == nil {
		suggs.AppendUnique(MissingMainFunc)
		return
	}
	cGroup := main.ChildByNodeType(astrav.NodeTypeCommentGroup)
	checkComment(cGroup, suggs, "function", "ShareWith")
}

var outputPart = regexp.MustCompile(`, one for me\.`)

func examConditional(pkg *astrav.Package, suggs sugg.Suggester) {
	main := pkg.FuncDeclByName("ShareWith")
	if main == nil {
		suggs.AppendUnique(MissingMainFunc)
		return
	}

	matches := outputPart.FindAllIndex(main.GetSource(), -1)
	if 1 < len(matches) {
		suggs.AppendUnique(MinimalConditional)
	}
}

func examGeneralizeNames(pkg *astrav.Package, suggs sugg.Suggester) {
	main := pkg.FuncDeclByName("ShareWith")
	if main == nil {
		suggs.AppendUnique(MissingMainFunc)
		return
	}

	contains := bytes.Contains(main.GetSource(), []byte("Alice"))
	if !contains {
		contains = bytes.Contains(main.GetSource(), []byte("Bob"))
	}
	if contains {
		suggs.AppendUnique(GeneralizeName)
	}
}

var commentStrings = map[string]struct {
	typeString       string
	stubString       string
	prefixString     string
	wrongCommentName string
}{
	"package": {
		typeString:       "Packages",
		stubString:       "should have a package comment",
		prefixString:     "Package %s ",
		wrongCommentName: "package `%s`",
	},
	"function": {
		typeString:       "Exported functions",
		stubString:       "should have a comment",
		prefixString:     "%s ",
		wrongCommentName: "function `%s`",
	},
}

// we only do this on the first exercise. Later we ask them to use golint.
func checkComment(cGroup astrav.Node, suggs sugg.Suggester, commentType, name string) {
	strPack := commentStrings[commentType]
	if cGroup == nil || len(cGroup.Children()) == 0 {
		suggs.AppendUnique(fmt.Sprintf("go.two_fer.missing_%s_comment", commentType))
		suggs.AppendUnique(CommentSection)
		return
	}

	comment, ok := cGroup.Children()[0].(*astrav.Comment)
	if !ok {
		suggs.ReportError(errors.New("expected comment in comment group"))
		return
	}
	cmt := strings.TrimSpace(strings.Replace(strings.Replace(comment.Text, "/*", "", 1), "//", "", 1))

	if strings.Contains(cmt, strPack.stubString) {
		suggs.AppendUnique(StubComments)
	} else if !strings.HasPrefix(cmt, fmt.Sprintf(strPack.prefixString, name)) {
		suggs.AppendUnique(fmt.Sprintf("go.two_fer.wrong_%s_comment", commentType))
		suggs.AppendUnique(CommentSection)
	}
}