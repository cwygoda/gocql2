package parser

import (
	"fmt"
	"strings"

	"github.com/cwygoda/gocql2/api"
)

type textParser struct {
	tokens []token
	cfg    ParseConfig
	pos    int
}

func parseText(input string, cfg ParseConfig) (api.Expression, error) {
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

func (p *textParser) parseExpression(depth int) (api.Expression, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	return p.parseOr(depth)
}

func (p *textParser) parseOr(depth int) (api.Expression, error) {
	left, err := p.parseAnd(depth + 1)
	if err != nil {
		return nil, err
	}
	args := []api.Expression{left}
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
	return &api.LogicalExpression{Op: api.LogicalOr, Args: args, Src: spanFrom(args[0], args[len(args)-1])}, nil
}

func (p *textParser) parseAnd(depth int) (api.Expression, error) {
	left, err := p.parseNot(depth + 1)
	if err != nil {
		return nil, err
	}
	args := []api.Expression{left}
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
	return &api.LogicalExpression{Op: api.LogicalAnd, Args: args, Src: spanFrom(args[0], args[len(args)-1])}, nil
}

func (p *textParser) parseNot(depth int) (api.Expression, error) {
	if tok, ok := p.consumeKeyword("NOT"); ok {
		expr, err := p.parseNot(depth + 1)
		if err != nil {
			return nil, err
		}
		return &api.LogicalExpression{Op: api.LogicalNot, Args: []api.Expression{expr}, Src: api.Span{Start: tok.span.Start, End: expr.Span().End}}, nil
	}
	return p.parsePrimaryExpression(depth + 1)
}

func (p *textParser) parsePrimaryExpression(depth int) (api.Expression, error) {
	if p.peek().kind == tokenKeyword {
		if op, ok := isSpatialPredicateOp(p.peek().text); ok {
			if !p.cfg.conformance.allowsSpatialPredicate(op) {
				return nil, p.errorHere("spatial predicate requires spatial conformance")
			}
			return p.parseSpatialPredicate(op, depth+1)
		}
		if op, ok := isTemporalPredicateOp(p.peek().text); ok {
			if !p.cfg.conformance.allowsTemporalPredicate(op) {
				return nil, p.errorHere("temporal predicate requires temporal-functions conformance")
			}
			return p.parseTemporalPredicate(op, depth+1)
		}
		if op, ok := isArrayPredicateOp(p.peek().text); ok {
			if !p.cfg.conformance.allowsArrayPredicate(op) {
				return nil, p.errorHere("array predicate requires array-functions conformance")
			}
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
		op := api.ComparisonOp(opTok.text)
		if err := validateComparisonOperands(op, left, right, api.LanguageText); err != nil {
			return nil, err
		}
		if err := validatePropertyPropertyConformance(p.cfg, api.LanguageText, left, right); err != nil {
			return nil, err
		}
		return &api.ComparisonExpression{Op: op, Left: left, Right: right, Src: api.Span{Start: left.Span().Start, End: right.Span().End}}, nil
	}
	if !p.cfg.conformance.arithmetic {
		if _, _, ok := p.peekArithmeticOperator(); ok {
			return nil, p.errorHere("arithmetic requires arithmetic conformance")
		}
	}

	not := p.matchKeyword("NOT")

	if p.matchKeyword("LIKE") {
		if !p.cfg.conformance.advancedComparisonOperators {
			return nil, p.errorPrevious("LIKE requires advanced-comparison-operators conformance")
		}
		if !isCharacterExpression(left) {
			return nil, parseError(api.LanguageText, left.Span().Start, "LIKE left operand must be a character expression")
		}
		pattern, err := p.parsePattern(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := validatePropertyPropertyConformance(p.cfg, api.LanguageText, left, pattern); err != nil {
			return nil, err
		}
		return &api.LikeExpression{Expr: left, Pattern: pattern, Not: not, Src: api.Span{Start: left.Span().Start, End: pattern.Span().End}}, nil
	}
	if p.matchKeyword("BETWEEN") {
		if !p.cfg.conformance.advancedComparisonOperators {
			return nil, p.errorPrevious("BETWEEN requires advanced-comparison-operators conformance")
		}
		if !isNumericExpression(left) {
			return nil, parseError(api.LanguageText, left.Span().Start, "BETWEEN operands must be numeric expressions")
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
		if err := validatePropertyPropertyConformance(p.cfg, api.LanguageText, left, lower, upper); err != nil {
			return nil, err
		}
		return &api.BetweenExpression{Expr: left, Lower: lower, Upper: upper, Not: not, Src: api.Span{Start: left.Span().Start, End: upper.Span().End}}, nil
	}
	if p.matchKeyword("IN") {
		if !p.cfg.conformance.advancedComparisonOperators {
			return nil, p.errorPrevious("IN requires advanced-comparison-operators conformance")
		}
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		var values []api.ScalarExpression
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
			return nil, parseError(api.LanguageText, end.span.Start, "IN list must not be empty")
		}
		if err := validateInOperands(left, values, api.LanguageText); err != nil {
			return nil, err
		}
		if err := validatePropertyPropertyConformance(p.cfg, api.LanguageText, append([]api.ScalarExpression{left}, values...)...); err != nil {
			return nil, err
		}
		return &api.InExpression{Expr: left, Values: values, Not: not, Src: api.Span{Start: left.Span().Start, End: end.span.End}}, nil
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

func (p *textParser) finishIsNull(operand api.Node) (*api.IsNullExpression, error) {
	notNull := p.matchKeyword("NOT")
	end, err := p.expectKeyword("NULL", "NULL")
	if err != nil {
		return nil, err
	}
	return &api.IsNullExpression{Expr: operand, Not: notNull, Src: api.Span{Start: operand.Span().Start, End: end.span.End}}, nil
}

func scalarAsExpression(scalar api.ScalarExpression) (api.Expression, bool) {
	if lit, ok := scalar.(*api.Literal); ok {
		if lit.Kind == api.LiteralBool {
			return lit, true
		}
		return nil, false
	}
	if fn, ok := scalar.(*api.FunctionCall); ok {
		if functionCallReturns(fn, api.FunctionTypeBoolean) {
			return fn, true
		}
		return nil, false
	}
	expr, ok := scalar.(api.Expression)
	return expr, ok
}

func (p *textParser) parsePattern(depth int) (api.ScalarExpression, error) {
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
		def, err := validateFunctionCall("casei", []api.Node{inner}, p.cfg, api.LanguageText, nameTok.span.Start)
		if err != nil {
			return nil, err
		}
		return &api.FunctionCall{Name: api.FunctionNameCaseI, Args: []api.Node{inner}, ReturnTypes: cloneFunctionTypes(def.Returns), Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
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
		def, err := validateFunctionCall("accenti", []api.Node{inner}, p.cfg, api.LanguageText, nameTok.span.Start)
		if err != nil {
			return nil, err
		}
		return &api.FunctionCall{Name: api.FunctionNameAccenti, Args: []api.Node{inner}, ReturnTypes: cloneFunctionTypes(def.Returns), Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
	}
	if !p.at(tokenString, "") {
		return nil, p.errorHere("LIKE pattern must be a character literal", "string literal", "CASEI", "ACCENTI")
	}
	return p.parseScalar(depth + 1)
}

func (p *textParser) parseNumericExpression(depth int) (api.ScalarExpression, error) {
	scalar, err := p.parseScalar(depth + 1)
	if err != nil {
		return nil, err
	}
	if !isNumericExpression(scalar) {
		return nil, parseError(api.LanguageText, scalar.Span().Start, "expected numeric expression", "number", "property", "function")
	}
	return scalar, nil
}

func (p *textParser) parseScalar(depth int) (api.ScalarExpression, error) {
	if p.cfg.conformance.arithmetic {
		return p.parseArithmeticExpression(depth+1, 0)
	}
	return p.parseScalarPrimary(depth + 1)
}

func (p *textParser) parseArithmeticExpression(depth, minPrecedence int) (api.ScalarExpression, error) {
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
			return nil, parseError(api.LanguageText, left.Span().Start, "arithmetic operands must be numeric expressions")
		}
		p.advance()

		nextMinPrecedence := precedence + 1
		if op == api.ArithmeticPow {
			nextMinPrecedence = precedence
		}
		right, err := p.parseArithmeticExpression(depth+1, nextMinPrecedence)
		if err != nil {
			return nil, err
		}
		if !isNumericExpression(right) {
			return nil, parseError(api.LanguageText, right.Span().Start, "arithmetic operands must be numeric expressions")
		}
		left = &api.ArithmeticExpression{Op: op, Left: left, Right: right, Src: api.Span{Start: left.Span().Start, End: right.Span().End}}
	}
}

func (p *textParser) parseScalarPrimary(depth int) (api.ScalarExpression, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	tok := p.peek()
	switch tok.kind {
	case tokenString:
		p.advance()
		return &api.Literal{Kind: api.LiteralString, Value: tok.text, Src: tok.span}, nil
	case tokenNumber:
		p.advance()
		value, err := canonicalNumber(tok.text)
		if err != nil {
			return nil, parseError(api.LanguageText, tok.span.Start, err.Error())
		}
		return &api.Literal{Kind: api.LiteralNumber, Value: value, Src: tok.span}, nil
	case tokenIdentifier, tokenQuotedIdentifier:
		p.advance()
		if p.match(tokenLParen, "") {
			return p.finishFunction(tok, depth+1)
		}
		return propertyRef(tok.text, tok.span, p.cfg, api.LanguageText, tok.span.Start)
	case tokenKeyword:
		if tok.text == "TRUE" || tok.text == "FALSE" {
			p.advance()
			return &api.Literal{Kind: api.LiteralBool, Value: tok.text == "TRUE", Src: tok.span}, nil
		}
		if tok.text == "DATE" || tok.text == "TIMESTAMP" {
			return p.parseTemporalInstantFunction(depth + 1)
		}
		if tok.text == "INTERVAL" {
			return nil, parseError(api.LanguageText, tok.span.Start, "INTERVAL is only allowed as a temporal operand", "temporal predicate", "IS NULL", "function argument")
		}
		if tok.text == "NULL" {
			return nil, parseError(api.LanguageText, tok.span.Start, "NULL is only allowed in IS NULL predicates", "scalar expression")
		}
		if _, ok := keywordFunctions[tok.text]; ok {
			p.advance()
			if !p.match(tokenLParen, "") {
				return nil, p.errorHere("expected function argument list", "(")
			}
			return p.finishFunction(tok, depth+1)
		}
		return nil, parseError(api.LanguageText, tok.span.Start, fmt.Sprintf("reserved keyword %q cannot be used as an unquoted property name or function", tok.text), "identifier", "quoted identifier")
	case tokenOperator:
		if tok.text != "-" && tok.text != "+" {
			return nil, p.errorHere("expected scalar expression", "property", "literal", "function")
		}
		if !p.cfg.conformance.arithmetic {
			return nil, p.errorHere("arithmetic requires arithmetic conformance")
		}
		p.advance()
		operand, err := p.parseScalarPrimary(depth + 1)
		if err != nil {
			return nil, err
		}
		if !isNumericExpression(operand) {
			return nil, parseError(api.LanguageText, operand.Span().Start, "arithmetic operands must be numeric expressions")
		}
		if tok.text == "+" {
			return operand, nil
		}
		zero := &api.Literal{Kind: api.LiteralNumber, Value: "0", Src: tok.span}
		return &api.ArithmeticExpression{Op: api.ArithmeticSub, Left: zero, Right: operand, Src: api.Span{Start: tok.span.Start, End: operand.Span().End}}, nil
	case tokenLParen:
		p.advance()
		var inner api.ScalarExpression
		var err error
		if p.cfg.conformance.arithmetic {
			inner, err = p.parseArithmeticExpression(depth+1, 0)
		} else {
			inner, err = p.parseScalar(depth + 1)
		}
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

func (p *textParser) peekArithmeticOperator() (api.ArithmeticOp, int, bool) {
	tok := p.peek()
	if tok.kind == tokenKeyword && tok.text == "DIV" {
		return api.ArithmeticIntDiv, 20, true
	}
	if tok.kind != tokenOperator {
		return "", 0, false
	}
	switch tok.text {
	case "+":
		return api.ArithmeticAdd, 10, true
	case "-":
		return api.ArithmeticSub, 10, true
	case "*":
		return api.ArithmeticMul, 20, true
	case "/":
		return api.ArithmeticDiv, 20, true
	case "%":
		return api.ArithmeticMod, 20, true
	case "^":
		return api.ArithmeticPow, 30, true
	default:
		return "", 0, false
	}
}

func (p *textParser) finishFunction(nameTok token, depth int) (*api.FunctionCall, error) {
	args := []api.Node{}
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
	def, err := validateFunctionCall(name, args, p.cfg, api.LanguageText, nameTok.span.Start)
	if err != nil {
		return nil, err
	}
	return &api.FunctionCall{Name: name, Args: args, ReturnTypes: cloneFunctionTypes(def.Returns), Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalPredicate(op api.TemporalPredicateOp, depth int) (*api.TemporalPredicateExpression, error) {
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
	if err := validateTemporalPredicateOperands(op, left, right, api.LanguageText); err != nil {
		return nil, err
	}
	return &api.TemporalPredicateExpression{Op: op, Left: left, Right: right, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalOperand(depth int) (api.Node, error) {
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

func (p *textParser) parseTemporalInstance(depth int) (api.Node, error) {
	if p.at(tokenKeyword, "DATE") || p.at(tokenKeyword, "TIMESTAMP") {
		return p.parseTemporalInstantFunction(depth + 1)
	}
	if p.at(tokenKeyword, "INTERVAL") {
		return p.parseTemporalInterval(depth + 1)
	}
	return nil, p.errorHere("expected temporal instance", "DATE", "TIMESTAMP", "INTERVAL")
}

func (p *textParser) parseTemporalInstantFunction(depth int) (*api.TemporalInstant, error) {
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
	kind := api.TemporalInstantDate
	if nameTok.text == "TIMESTAMP" {
		kind = api.TemporalInstantTimestamp
	}
	if kind == api.TemporalInstantDate {
		if err := validateDateLiteral(valueTok.text); err != nil {
			return nil, parseError(api.LanguageText, valueTok.span.Start, err.Error())
		}
	} else if err := validateTimestampLiteral(valueTok.text); err != nil {
		return nil, parseError(api.LanguageText, valueTok.span.Start, err.Error())
	}
	return &api.TemporalInstant{Kind: kind, Value: valueTok.text, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalInterval(depth int) (*api.TemporalInterval, error) {
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
	if err := validateTemporalIntervalOperands(start, endNode, api.LanguageText); err != nil {
		return nil, err
	}
	return &api.TemporalInterval{Start: start, End: endNode, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTemporalIntervalEndpoint(depth int) (api.Node, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	if p.at(tokenString, "") {
		tok := p.advance()
		if tok.text == ".." {
			return &api.TemporalUnbounded{Src: tok.span}, nil
		}
		kind, err := temporalInstantKindFromString(tok.text)
		if err != nil {
			return nil, parseError(api.LanguageText, tok.span.Start, err.Error())
		}
		return &api.TemporalInstant{Kind: kind, Value: tok.text, Src: tok.span}, nil
	}
	if p.peek().kind == tokenIdentifier || p.peek().kind == tokenQuotedIdentifier {
		return p.parseScalarPrimary(depth + 1)
	}
	return nil, p.errorHere("expected interval endpoint", "date string", "timestamp string", "..", "property", "function")
}

func (p *textParser) parseArrayPredicate(op api.ArrayPredicateOp, depth int) (*api.ArrayPredicateExpression, error) {
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
	if err := validateArrayPredicateOperands(left, right, api.LanguageText); err != nil {
		return nil, err
	}
	return &api.ArrayPredicateExpression{Op: op, Left: left, Right: right, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseArrayOperand(depth int) (api.Node, error) {
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

func (p *textParser) parseArrayLiteral(depth int) (*api.ArrayLiteral, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	start, err := p.expect(tokenLParen, "opening parenthesis")
	if err != nil {
		return nil, err
	}
	values := []api.Node{}
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
	return &api.ArrayLiteral{Values: values, Src: api.Span{Start: start.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseArrayElement(depth int) (api.Node, error) {
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

func (p *textParser) parseFunctionArg(depth int) (api.Node, error) {
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

func spanFrom(first, last api.Node) api.Span {
	return api.Span{Start: first.Span().Start, End: last.Span().End}
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

func (p *textParser) errorHere(message string, expected ...string) *api.ParseError {
	return parseError(api.LanguageText, p.peek().span.Start, message, expected...)
}

func (p *textParser) errorPrevious(message string, expected ...string) *api.ParseError {
	return parseError(api.LanguageText, p.previous().span.Start, message, expected...)
}
