package gocql2

import (
	"fmt"
	"strings"
)

type textParser struct {
	tokens []token
	cfg    ParseConfig
	pos    int
}

func parseText(input string, cfg ParseConfig) (Expression, error) {
	cfg = applyParseConfigDefaults(cfg)
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
	if p.peek().kind == tokenKeyword {
		if op, ok := isSpatialPredicateOp(p.peek().text); ok {
			return p.parseSpatialPredicate(op, depth+1)
		}
		if op, ok := isTemporalPredicateOp(p.peek().text); ok {
			return p.parseTemporalPredicate(op, depth+1)
		}
		if op, ok := isArrayPredicateOp(p.peek().text); ok {
			return p.parseArrayPredicate(op, depth+1)
		}
		if isGeometryKeyword(p.peek()) {
			operand, err := p.parseTextGeometryLiteral(depth + 1)
			if err != nil {
				return nil, err
			}
			if p.matchKeyword("IS") {
				return p.finishIsNull(operand)
			}
			return nil, p.errorHere("expected predicate operator", "IS")
		}
		if p.peek().text == "INTERVAL" {
			operand, err := p.parseTemporalInstance(depth + 1)
			if err != nil {
				return nil, err
			}
			if p.matchKeyword("IS") {
				return p.finishIsNull(operand)
			}
			return nil, p.errorHere("expected predicate operator", "IS")
		}
	}

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
		op := ComparisonOp(opTok.text)
		if err := validateComparisonOperands(op, left, right, LanguageText); err != nil {
			return nil, err
		}
		return &ComparisonExpression{Op: op, Left: left, Right: right, Src: Span{Start: left.Span().Start, End: right.Span().End}}, nil
	}

	not := p.matchKeyword("NOT")

	if p.matchKeyword("LIKE") {
		if !isCharacterExpression(left) {
			return nil, parseError(LanguageText, left.Span().Start, "LIKE left operand must be a character expression")
		}
		pattern, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, err
		}
		return &LikeExpression{Expr: left, Pattern: pattern, Not: not, Src: Span{Start: left.Span().Start, End: pattern.Span().End}}, nil
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
		if err := validateInOperands(left, values, LanguageText); err != nil {
			return nil, err
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
	if fn, ok := scalar.(*FunctionCall); ok {
		if functionCallReturns(fn, FunctionTypeBoolean) {
			return fn, true
		}
		return nil, false
	}
	expr, ok := scalar.(Expression)
	return expr, ok
}

func (p *textParser) parsePattern(depth int) (ScalarExpression, error) {
	if nameTok, ok := p.consumeKeyword("CASEI"); ok {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		inner, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, err
		}
		end, err := p.expect(tokenRParen, "closing parenthesis")
		if err != nil {
			return nil, err
		}
		def, err := validateFunctionCall("casei", []Node{inner}, p.cfg, LanguageText, nameTok.span.Start)
		if err != nil {
			return nil, err
		}
		return &FunctionCall{Name: FunctionNameCaseI, Args: []Node{inner}, ReturnTypes: cloneFunctionTypes(def.Returns), Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
	}
	if nameTok, ok := p.consumeKeyword("ACCENTI"); ok {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		inner, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, err
		}
		end, err := p.expect(tokenRParen, "closing parenthesis")
		if err != nil {
			return nil, err
		}
		def, err := validateFunctionCall("accenti", []Node{inner}, p.cfg, LanguageText, nameTok.span.Start)
		if err != nil {
			return nil, err
		}
		return &FunctionCall{Name: FunctionNameAccenti, Args: []Node{inner}, ReturnTypes: cloneFunctionTypes(def.Returns), Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
	}
	if !p.at(tokenString, "") {
		return nil, p.errorHere("LIKE pattern must be a character literal", "string literal", "CASEI", "ACCENTI")
	}
	return p.parseScalar(depth + 1)
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
		return propertyRef(tok.text, tok.span, p.cfg, LanguageText, tok.span.Start)
	case tokenKeyword:
		if tok.text == "TRUE" || tok.text == "FALSE" {
			p.advance()
			return &Literal{Kind: LiteralBool, Value: tok.text == "TRUE", Src: tok.span}, nil
		}
		if tok.text == "DATE" || tok.text == "TIMESTAMP" {
			return p.parseTemporalInstantFunction(depth + 1)
		}
		if tok.text == "INTERVAL" {
			return nil, parseError(LanguageText, tok.span.Start, "INTERVAL is only allowed as a temporal operand", "temporal predicate", "IS NULL", "function argument")
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
	name := strings.ToLower(nameTok.text)
	def, err := validateFunctionCall(name, args, p.cfg, LanguageText, nameTok.span.Start)
	if err != nil {
		return nil, err
	}
	return &FunctionCall{Name: name, Args: args, ReturnTypes: cloneFunctionTypes(def.Returns), Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalPredicate(op TemporalPredicateOp, depth int) (*TemporalPredicateExpression, error) {
	nameTok := p.advance()
	if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
		return nil, err
	}
	left, err := p.parseTemporalOperand(depth + 1)
	if err != nil {
		return nil, err
	}
	if _, expectErr := p.expect(tokenComma, "comma"); expectErr != nil {
		return nil, expectErr
	}
	right, err := p.parseTemporalOperand(depth + 1)
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	if err := validateTemporalPredicateOperands(op, left, right, LanguageText); err != nil {
		return nil, err
	}
	return &TemporalPredicateExpression{Op: op, Left: left, Right: right, Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalOperand(depth int) (Node, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	if p.at(tokenKeyword, "DATE") || p.at(tokenKeyword, "TIMESTAMP") || p.at(tokenKeyword, "INTERVAL") {
		return p.parseTemporalInstance(depth + 1)
	}
	node, err := p.parseScalarPrimary(depth + 1)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (p *textParser) parseTemporalInstance(depth int) (Node, error) {
	if p.at(tokenKeyword, "DATE") || p.at(tokenKeyword, "TIMESTAMP") {
		return p.parseTemporalInstantFunction(depth + 1)
	}
	if p.at(tokenKeyword, "INTERVAL") {
		return p.parseTemporalInterval(depth + 1)
	}
	return nil, p.errorHere("expected temporal instance", "DATE", "TIMESTAMP", "INTERVAL")
}

func (p *textParser) parseTemporalInstantFunction(depth int) (*TemporalInstant, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	nameTok := p.advance()
	if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
		return nil, err
	}
	valueTok, err := p.expect(tokenString, "temporal literal string")
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	kind := TemporalInstantDate
	if nameTok.text == "TIMESTAMP" {
		kind = TemporalInstantTimestamp
	}
	if kind == TemporalInstantDate {
		if err := validateDateLiteral(valueTok.text); err != nil {
			return nil, parseError(LanguageText, valueTok.span.Start, err.Error())
		}
	} else if err := validateTimestampLiteral(valueTok.text, p.cfg.StrictTimestampUTC); err != nil {
		return nil, parseError(LanguageText, valueTok.span.Start, err.Error())
	}
	return &TemporalInstant{Kind: kind, Value: valueTok.text, Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalInterval(depth int) (*TemporalInterval, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	nameTok, _ := p.consumeKeyword("INTERVAL")
	if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
		return nil, err
	}
	start, err := p.parseTemporalIntervalEndpoint(depth + 1)
	if err != nil {
		return nil, err
	}
	if _, expectErr := p.expect(tokenComma, "comma"); expectErr != nil {
		return nil, expectErr
	}
	endNode, err := p.parseTemporalIntervalEndpoint(depth + 1)
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	if err := validateTemporalIntervalOperands(start, endNode, LanguageText); err != nil {
		return nil, err
	}
	return &TemporalInterval{Start: start, End: endNode, Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalIntervalEndpoint(depth int) (Node, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	if p.at(tokenString, "") {
		tok := p.advance()
		if tok.text == ".." {
			return &TemporalUnbounded{Src: tok.span}, nil
		}
		kind, err := temporalInstantKindFromString(tok.text, p.cfg.StrictTimestampUTC)
		if err != nil {
			return nil, parseError(LanguageText, tok.span.Start, err.Error())
		}
		return &TemporalInstant{Kind: kind, Value: tok.text, Src: tok.span}, nil
	}
	if p.peek().kind == tokenIdentifier || p.peek().kind == tokenQuotedIdentifier {
		return p.parseScalarPrimary(depth + 1)
	}
	return nil, p.errorHere("expected interval endpoint", "date string", "timestamp string", "..", "property", "function")
}

func (p *textParser) parseArrayPredicate(op ArrayPredicateOp, depth int) (*ArrayPredicateExpression, error) {
	nameTok := p.advance()
	if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
		return nil, err
	}
	left, err := p.parseArrayOperand(depth + 1)
	if err != nil {
		return nil, err
	}
	if _, expectErr := p.expect(tokenComma, "comma"); expectErr != nil {
		return nil, expectErr
	}
	right, err := p.parseArrayOperand(depth + 1)
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	if err := validateArrayPredicateOperands(left, right, LanguageText); err != nil {
		return nil, err
	}
	return &ArrayPredicateExpression{Op: op, Left: left, Right: right, Src: Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseArrayOperand(depth int) (Node, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	if p.at(tokenLParen, "") {
		return p.parseArrayLiteral(depth + 1)
	}

	node, err := p.parseScalarPrimary(depth + 1)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (p *textParser) parseArrayLiteral(depth int) (*ArrayLiteral, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	start, err := p.expect(tokenLParen, "opening parenthesis")
	if err != nil {
		return nil, err
	}
	values := []Node{}
	if !p.at(tokenRParen, "") {
		for {
			value, valueErr := p.parseArrayElement(depth + 1)
			if valueErr != nil {
				return nil, valueErr
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
	return &ArrayLiteral{Values: values, Src: Span{Start: start.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseArrayElement(depth int) (Node, error) {
	if p.at(tokenLParen, "") {
		return p.parseArrayLiteral(depth + 1)
	}
	start := p.pos
	if expr, err := p.parseExpression(depth + 1); err == nil {
		return expr, nil
	}
	p.pos = start
	if p.at(tokenKeyword, "INTERVAL") {
		return p.parseTemporalInstance(depth + 1)
	}
	if isGeometryKeyword(p.peek()) {
		return p.parseTextGeometryLiteral(depth + 1)
	}
	return p.parseScalar(depth + 1)
}

func (p *textParser) parseFunctionArg(depth int) (Node, error) {
	start := p.pos
	if expr, err := p.parseExpression(depth + 1); err == nil {
		return expr, nil
	}
	p.pos = start
	if p.at(tokenKeyword, "INTERVAL") {
		return p.parseTemporalInstance(depth + 1)
	}
	if isGeometryKeyword(p.peek()) {
		return p.parseTextGeometryLiteral(depth + 1)
	}
	if scalar, err := p.parseScalar(depth + 1); err == nil {
		return scalar, nil
	}
	p.pos = start
	if p.at(tokenLParen, "") {
		return p.parseArrayLiteral(depth + 1)
	}
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
