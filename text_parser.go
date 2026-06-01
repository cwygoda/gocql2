package gocql2

import (
	"fmt"
	"strings"
)

type textParser struct {
	tokens []token
	pos    int
	cfg    ParseConfig
}

func parseText(input string, cfg ParseConfig) (Expression, error) {
	tokens, err := lexText(input)
	if err != nil {
		return nil, err
	}
	p := &textParser{tokens: tokens, cfg: cfg}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if !p.at(tokenEOF, "") {
		return nil, p.errorHere("unexpected trailing input", "end of input")
	}
	return expr, nil
}

func (p *textParser) parseExpression(depth int) (Expression, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	return p.parseOr(depth)
}

func (p *textParser) parseOr(depth int) (Expression, error) {
	left, err := p.parseAnd(depth + 1)
	if err != nil {
		return nil, err
	}
	args := []Expression{left}
	for p.matchKeyword("OR") {
		right, err := p.parseAnd(depth + 1)
		if err != nil {
			return nil, err
		}
		args = append(args, right)
	}
	if len(args) == 1 {
		return left, nil
	}
	return &LogicalExpression{Op: LogicalOr, Args: args, Src: spanFrom(args[0], args[len(args)-1])}, nil
}

func (p *textParser) parseAnd(depth int) (Expression, error) {
	left, err := p.parseNot(depth + 1)
	if err != nil {
		return nil, err
	}
	args := []Expression{left}
	for p.matchKeyword("AND") {
		right, err := p.parseNot(depth + 1)
		if err != nil {
			return nil, err
		}
		args = append(args, right)
	}
	if len(args) == 1 {
		return left, nil
	}
	return &LogicalExpression{Op: LogicalAnd, Args: args, Src: spanFrom(args[0], args[len(args)-1])}, nil
}

func (p *textParser) parseNot(depth int) (Expression, error) {
	if tok, ok := p.consumeKeyword("NOT"); ok {
		expr, err := p.parseNot(depth + 1)
		if err != nil {
			return nil, err
		}
		return &LogicalExpression{Op: LogicalNot, Args: []Expression{expr}, Src: Span{Start: tok.span.Start, End: expr.Span().End}}, nil
	}
	return p.parsePrimaryExpression(depth + 1)
}

func (p *textParser) parsePrimaryExpression(depth int) (Expression, error) {
	if p.at(tokenLParen, "") {
		startPos := p.pos
		p.advance()
		expr, err := p.parseExpression(depth + 1)
		if err == nil {
			if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
				return nil, err
			}
			if p.matchKeyword("IS") {
				return p.finishIsNull(expr)
			}
			return expr, nil
		}
		p.pos = startPos
	}

	left, err := p.parseScalar(depth + 1)
	if err != nil {
		return nil, err
	}

	if opTok, ok := p.consumeComparisonOperator(); ok {
		right, err := p.parseScalar(depth + 1)
		if err != nil {
			return nil, err
		}
		return &ComparisonExpression{Op: ComparisonOp(opTok.text), Left: left, Right: right, Src: Span{Start: left.Span().Start, End: right.Span().End}}, nil
	}

	not := p.matchKeyword("NOT")

	if p.matchKeyword("LIKE") {
		if !isCharacterExpression(left) {
			return nil, parseError(LanguageText, left.Span().Start, "LIKE left operand must be a character expression")
		}
		pattern, modifier, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, err
		}
		return &LikeExpression{Expr: left, Pattern: pattern, Not: not, Modifier: modifier, Src: Span{Start: left.Span().Start, End: pattern.Span().End}}, nil
	}
	if p.matchKeyword("BETWEEN") {
		if !isNumericExpression(left) {
			return nil, parseError(LanguageText, left.Span().Start, "BETWEEN operands must be numeric expressions")
		}
		lower, err := p.parseNumericExpression(depth + 1)
		if err != nil {
			return nil, err
		}
		if _, expectErr := p.expectKeyword("AND", "AND"); expectErr != nil {
			return nil, expectErr
		}
		upper, err := p.parseNumericExpression(depth + 1)
		if err != nil {
			return nil, err
		}
		return &BetweenExpression{Expr: left, Lower: lower, Upper: upper, Not: not, Src: Span{Start: left.Span().Start, End: upper.Span().End}}, nil
	}
	if p.matchKeyword("IN") {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		var values []ScalarExpression
		if !p.at(tokenRParen, "") {
			for {
				value, err := p.parseScalar(depth + 1)
				if err != nil {
					return nil, err
				}
				values = append(values, value)
				if !p.match(tokenComma, "") {
					break
				}
			}
		}
		end, err := p.expect(tokenRParen, "closing parenthesis")
		if err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, parseError(LanguageText, end.span.Start, "IN list must not be empty")
		}
		return &InExpression{Expr: left, Values: values, Not: not, Src: Span{Start: left.Span().Start, End: end.span.End}}, nil
	}
	if not {
		return nil, p.errorPrevious("expected LIKE, BETWEEN, or IN after NOT", "LIKE", "BETWEEN", "IN")
	}

	if p.matchKeyword("IS") {
		return p.finishIsNull(left)
	}

	if expr, ok := scalarAsExpression(left); ok {
		return expr, nil
	}
	return nil, p.errorHere("expected predicate operator", "=", "<>", "<", ">", "<=", ">=", "LIKE", "BETWEEN", "IN", "IS")
}

func (p *textParser) finishIsNull(operand Node) (*IsNullExpression, error) {
	notNull := p.matchKeyword("NOT")
	end, err := p.expectKeyword("NULL", "NULL")
	if err != nil {
		return nil, err
	}
	return &IsNullExpression{Expr: operand, Not: notNull, Src: Span{Start: operand.Span().Start, End: end.span.End}}, nil
}

func scalarAsExpression(scalar ScalarExpression) (Expression, bool) {
	if lit, ok := scalar.(*Literal); ok {
		if lit.Kind == LiteralBool {
			return lit, true
		}
		return nil, false
	}
	expr, ok := scalar.(Expression)
	return expr, ok
}

func (p *textParser) parsePattern(depth int) (ScalarExpression, string, error) {
	if _, ok := p.consumeKeyword("CASEI"); ok {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, "", err
		}
		inner, _, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, "", err
		}
		if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
			return nil, "", err
		}
		return inner, "casei", nil
	}
	if _, ok := p.consumeKeyword("ACCENTI"); ok {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, "", err
		}
		inner, _, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, "", err
		}
		if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
			return nil, "", err
		}
		return inner, "accenti", nil
	}
	if !p.at(tokenString, "") {
		return nil, "", p.errorHere("LIKE pattern must be a character literal", "string literal", "CASEI", "ACCENTI")
	}
	scalar, err := p.parseScalar(depth + 1)
	return scalar, "", err
}

func (p *textParser) parseNumericExpression(depth int) (ScalarExpression, error) {
	scalar, err := p.parseScalar(depth + 1)
	if err != nil {
		return nil, err
	}
	if !isNumericExpression(scalar) {
		return nil, parseError(LanguageText, scalar.Span().Start, "expected numeric expression", "number", "property", "function")
	}
	return scalar, nil
}

func isCharacterExpression(scalar ScalarExpression) bool {
	switch value := scalar.(type) {
	case *Literal:
		return value.Kind == LiteralString
	case *PropertyRef, *FunctionCall:
		return true
	default:
		return false
	}
}

func isNumericExpression(scalar ScalarExpression) bool {
	switch value := scalar.(type) {
	case *Literal:
		return value.Kind == LiteralNumber
	case *PropertyRef, *FunctionCall, *ArithmeticExpression:
		return true
	default:
		return false
	}
}

func (p *textParser) parseScalar(depth int) (ScalarExpression, error) {
	return p.parseArithmeticExpression(depth+1, 0)
}

func (p *textParser) parseArithmeticExpression(depth, minPrecedence int) (ScalarExpression, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}

	left, err := p.parseScalarPrimary(depth + 1)
	if err != nil {
		return nil, err
	}

	for {
		op, precedence, ok := p.peekArithmeticOperator()
		if !ok || precedence < minPrecedence {
			return left, nil
		}
		if !isNumericExpression(left) {
			return nil, parseError(LanguageText, left.Span().Start, "arithmetic operands must be numeric expressions")
		}
		p.advance()

		nextMinPrecedence := precedence + 1
		if op == ArithmeticPow {
			nextMinPrecedence = precedence
		}
		right, err := p.parseArithmeticExpression(depth+1, nextMinPrecedence)
		if err != nil {
			return nil, err
		}
		if !isNumericExpression(right) {
			return nil, parseError(LanguageText, right.Span().Start, "arithmetic operands must be numeric expressions")
		}
		left = &ArithmeticExpression{Op: op, Left: left, Right: right, Src: Span{Start: left.Span().Start, End: right.Span().End}}
	}
}

func (p *textParser) parseScalarPrimary(depth int) (ScalarExpression, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	tok := p.peek()
	switch tok.kind {
	case tokenString:
		p.advance()
		return &Literal{Kind: LiteralString, Value: tok.text, Src: tok.span}, nil
	case tokenNumber:
		p.advance()
		value, err := canonicalNumber(tok.text)
		if err != nil {
			return nil, parseError(LanguageText, tok.span.Start, err.Error())
		}
		return &Literal{Kind: LiteralNumber, Value: value, Src: tok.span}, nil
	case tokenIdentifier, tokenQuotedIdentifier:
		p.advance()
		if p.match(tokenLParen, "") {
			return p.finishFunction(tok, depth+1)
		}
		return &PropertyRef{Name: tok.text, Src: tok.span}, nil
	case tokenKeyword:
		if tok.text == "TRUE" || tok.text == "FALSE" {
			p.advance()
			return &Literal{Kind: LiteralBool, Value: tok.text == "TRUE", Src: tok.span}, nil
		}
		if tok.text == "NULL" {
			return nil, parseError(LanguageText, tok.span.Start, "NULL is only allowed in IS NULL predicates", "scalar expression")
		}
		if _, ok := keywordFunctions[tok.text]; ok {
			p.advance()
			if !p.match(tokenLParen, "") {
				return nil, p.errorHere("expected function argument list", "(")
			}
			return p.finishFunction(tok, depth+1)
		}
		return nil, parseError(LanguageText, tok.span.Start, fmt.Sprintf("reserved keyword %q cannot be used as an unquoted property name or function", tok.text), "identifier", "quoted identifier")
	case tokenOperator:
		if tok.text != "-" {
			return nil, p.errorHere("expected scalar expression", "property", "literal", "function")
		}
		p.advance()
		operand, err := p.parseScalarPrimary(depth + 1)
		if err != nil {
			return nil, err
		}
		if !isNumericExpression(operand) {
			return nil, parseError(LanguageText, operand.Span().Start, "arithmetic operands must be numeric expressions")
		}
		zero := &Literal{Kind: LiteralNumber, Value: "0", Src: tok.span}
		return &ArithmeticExpression{Op: ArithmeticSub, Left: zero, Right: operand, Src: Span{Start: tok.span.Start, End: operand.Span().End}}, nil
	case tokenLParen:
		p.advance()
		inner, err := p.parseArithmeticExpression(depth+1, 0)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
			return nil, err
		}
		return inner, nil
	default:
		return nil, p.errorHere("expected scalar expression", "property", "literal", "function")
	}
}

func (p *textParser) peekArithmeticOperator() (ArithmeticOp, int, bool) {
	tok := p.peek()
	if tok.kind == tokenKeyword && tok.text == "DIV" {
		return ArithmeticIntDiv, 20, true
	}
	if tok.kind != tokenOperator {
		return "", 0, false
	}
	switch tok.text {
	case "+":
		return ArithmeticAdd, 10, true
	case "-":
		return ArithmeticSub, 10, true
	case "*":
		return ArithmeticMul, 20, true
	case "/":
		return ArithmeticDiv, 20, true
	case "%":
		return ArithmeticMod, 20, true
	case "^":
		return ArithmeticPow, 30, true
	default:
		return "", 0, false
	}
}

func (p *textParser) finishFunction(nameTok token, depth int) (*FunctionCall, error) {
	args := []Node{}
	if !p.at(tokenRParen, "") {
		for {
			arg, err := p.parseFunctionArg(depth + 1)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if !p.match(tokenComma, "") {
				break
			}
		}
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &FunctionCall{Name: strings.ToLower(nameTok.text), Args: args, Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseFunctionArg(depth int) (Node, error) {
	start := p.pos
	if expr, err := p.parseExpression(depth + 1); err == nil {
		return expr, nil
	}
	p.pos = start
	return p.parseScalar(depth + 1)
}

func (p *textParser) consumeComparisonOperator() (token, bool) {
	if p.peek().kind != tokenOperator {
		return token{}, false
	}
	switch p.peek().text {
	case "=", "<>", "<", ">", "<=", ">=":
		return p.advance(), true
	default:
		return token{}, false
	}
}

func spanFrom(first, last Node) Span {
	return Span{Start: first.Span().Start, End: last.Span().End}
}

func (p *textParser) peek() token { return p.tokens[p.pos] }

func (p *textParser) previous() token { return p.tokens[p.pos-1] }

func (p *textParser) advance() token {
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return p.tokens[p.pos-1]
}

func (p *textParser) at(kind tokenKind, text string) bool {
	tok := p.peek()
	if tok.kind != kind {
		return false
	}
	return text == "" || tok.text == text
}

func (p *textParser) match(kind tokenKind, text string) bool {
	if !p.at(kind, text) {
		return false
	}
	p.advance()
	return true
}

func (p *textParser) matchKeyword(keyword string) bool {
	return p.match(tokenKeyword, keyword)
}

func (p *textParser) consumeKeyword(keyword string) (token, bool) {
	if !p.at(tokenKeyword, keyword) {
		return token{}, false
	}
	return p.advance(), true
}

func (p *textParser) expect(kind tokenKind, expected string) (token, error) {
	if !p.at(kind, "") {
		return token{}, p.errorHere("unexpected token", expected)
	}
	return p.advance(), nil
}

func (p *textParser) expectKeyword(keyword, expected string) (token, error) {
	if !p.at(tokenKeyword, keyword) {
		return token{}, p.errorHere("unexpected token", expected)
	}
	return p.advance(), nil
}

func (p *textParser) errorHere(message string, expected ...string) *ParseError {
	return parseError(LanguageText, p.peek().span.Start, message, expected...)
}

func (p *textParser) errorPrevious(message string, expected ...string) *ParseError {
	return parseError(LanguageText, p.previous().span.Start, message, expected...)
}
