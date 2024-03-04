package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// token type
	SYMBOL     = "symbol"
	KEYWORD    = "keyword"
	IDENTIFIER = "identifier"
	INT_CONST  = "int_const"
	STR_CONST  = "str_const"

	// variable scope
	STATIC   = "static"
	FIELD    = "field"
	ARGUMENT = "argument"
	LOCAL    = "local"
	NONE     = "none"
)

func main() {
	var dir string
	var fileNames []string

	if len(os.Args) < 2 {
		executable, err := os.Executable()
		if err != nil {
			return
		}
		dir = filepath.Dir(executable)
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name(), ".jack") {
				fileNames = append(fileNames, entry.Name())
			}
		}
	} else {
		asm := os.Args[1]
		fileInfo, err := os.Stat(asm)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(asm)
			if err != nil {
				fmt.Printf("%v", err)
				return
			}
			for _, entry := range entries {
				if strings.Contains(entry.Name(), ".jack") {
					fileNames = append(fileNames, entry.Name())
				}
			}
			dir = asm
		} else {
			fileNames = append(fileNames, fileInfo.Name())
			dir = filepath.Dir(asm)
		}
	}

	engine := NewCompilationEngine()

	// generate token file
	for _, fileName := range fileNames {
		tokenizer := NewJackTokenizer(filepath.Join(dir, fileName))
		err := tokenizer.read()
		if err != nil {
			return
		}
		engine.setTokens(tokenizer.tokens)
		names := strings.Split(fileName, ".")

		engine.index = 0
		engine.vm = nil
		err = engine.compileClass()
		if err != nil {
			return
		}

		err = os.WriteFile(filepath.Join(dir, names[0]+".vm"), engine.vm, 0644)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		if len(engine.vm) > 0 {
			fmt.Printf("generate vm file %s successfully\n", names[0]+".vm")
		}
	}
}

type JackTokenizer struct {
	file   string
	tokens []string
}

func NewJackTokenizer(file string) *JackTokenizer {
	return &JackTokenizer{file: file}
}

func (j *JackTokenizer) read() error {
	var comment bool
	content, err := os.ReadFile(j.file)
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if index := strings.Index(line, "//"); index > -1 {
			line = line[:index]
		}
		line = strings.TrimSpace(line)
		if len(line) < 1 || (len(line) > 2 && line[0:2] == "//") {
			continue
		}
		if comment {
			if strings.Contains(line, "*/") {
				comment = false
			}
			continue
		}

		if len(line) > 1 && line[0:2] == "/*" {
			comment = true
			if strings.Contains(line, "*/") {
				comment = false
			}
			continue
		}

		var buf []byte
		var str bool
		var text bool
		var cons bool
		for i := 0; i < len(line); i++ {
			if str {
				if line[i] == '"' {
					str = false
					buf = append(buf, line[i])
					j.addToken(string(buf))
					buf = nil
				} else {
					buf = append(buf, line[i])
				}
				continue
			} else if text {
				if line[i] == ' ' || isSymbol(line[i]) {
					text = false
					j.addToken(string(buf))
					buf = nil
					if line[i] != ' ' {
						j.addToken(string(line[i]))
					}
				} else {
					buf = append(buf, line[i])
				}
				continue
			} else if cons {
				if line[i] == ' ' || isSymbol(line[i]) {
					cons = false
					j.addToken(string(buf))
					buf = nil
					if line[i] != ' ' {
						j.addToken(string(line[i]))
					}
				} else {
					buf = append(buf, line[i])
				}
				continue
			}

			if isSymbol(line[i]) {
				j.addToken(string(line[i]))
			} else if isNumber(line[i]) {
				cons = true
				buf = append(buf, line[i])
			} else if line[i] == '"' {
				str = true
				buf = append(buf, line[i])
			} else if line[i] != ' ' {
				text = true
				buf = append(buf, line[i])
			}
		}
	}

	return nil
}

func isSymbol(c byte) bool {
	return c == '{' || c == '}' || c == '(' || c == ')' || c == '[' || c == ']' || c == '.' || c == ',' || c == ';' || c == '+' || c == '-' || c == '*' || c == '/' || c == '&' || c == '<' || c == '>' || c == '|' || c == '=' || c == '~'
}

func isKeyword(s string) bool {
	return s == "class" || s == "constructor" || s == "function" || s == "method" || s == "field" || s == "static" || s == "var" || s == "int" || s == "char" || s == "boolean" || s == "void" || s == "true" || s == "false" || s == "null" || s == "this" || s == "let" || s == "do" || s == "if" || s == "else" || s == "while" || s == "return"
}

func isNumber(c byte) bool {
	return c >= '0' && c <= '9'
}

func isStrConst(c byte) bool {
	return c == '"'
}

func (j *JackTokenizer) addToken(s string) {
	if len(s) > 0 {
		j.tokens = append(j.tokens, s)
	}
}

func (j *JackTokenizer) tokenType(s string) string {
	if isSymbol(s[0]) {
		return SYMBOL
	} else if isKeyword(s) {
		return KEYWORD
	} else if isStrConst(s[0]) {
		return STR_CONST
	} else if isNumber(s[0]) {
		return INT_CONST
	} else {
		return IDENTIFIER
	}
}

func tokenType(s string) string {
	if isSymbol(s[0]) {
		return SYMBOL
	} else if isKeyword(s) {
		return KEYWORD
	} else if isStrConst(s[0]) {
		return STR_CONST
	} else if isNumber(s[0]) {
		return INT_CONST
	} else {
		return IDENTIFIER
	}
}

func (j *JackTokenizer) keyWord(s string) string {
	return strings.ToUpper(s)
}

func (j *JackTokenizer) symbol(s string) string {
	return s
}

func (j *JackTokenizer) identifier(s string) string {
	return s
}

func (j *JackTokenizer) intVal(s string) int {
	atoi, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return atoi
}

func (j *JackTokenizer) strVal(s string) string {
	return s[1 : len(s)-1]
}

// 是针对 program structure, statement, expression 进行处理
type CompilationEngine struct {
	index           int
	vm              []byte
	tokens          []string
	classTable      *ClassSymbolTable
	subroutineTable *SubroutineSymbolTable
}

func NewCompilationEngine() *CompilationEngine {
	return &CompilationEngine{
		classTable:      NewClassSymbolTable(),
		subroutineTable: NewSubroutineSymbolTable(),
	}
}

func (c *CompilationEngine) curToken() string {
	return c.tokens[c.index]
}

func (c *CompilationEngine) identifierProcess(expected string, tokenType string, integerType string, level string, usage string) int {
	var index int
	if c.curToken() == expected {
		if level == STATIC || level == FIELD {
			if usage == "declare" {
				c.classTable.define(c.curToken(), integerType, level)
			}
			index = c.classTable.indexOf(expected)
		} else if level == ARGUMENT || level == LOCAL {
			if usage == "declare" {
				c.subroutineTable.define(c.curToken(), integerType, level)
			}
			index = c.subroutineTable.indexOf(expected)
		}
	} else {
		fmt.Printf("expected %s real %s\n", expected, c.curToken())
		// for simplicity
		os.Exit(1)
	}
	c.index += 1
	return index
}

func (c *CompilationEngine) process(expected string) {
	if c.curToken() != expected {
		fmt.Printf("expected %s real %s\n", expected, c.curToken())
		// for simplicity
		os.Exit(1)
	}
	c.index += 1
}

func (c *CompilationEngine) setTokens(tokens []string) {
	c.tokens = tokens
}

func (c *CompilationEngine) compileClass() error {
	c.classTable.reset()
	c.subroutineTable.reset()

	c.process("class")
	c.classTable.setName(c.curToken())
	c.process(c.curToken())
	c.process("{")

	for c.curToken() == "static" || c.curToken() == "field" {
		err := c.compileClassVarDec()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	for c.curToken() == "constructor" || c.curToken() == "function" || c.curToken() == "method" {
		err := c.compileSubroutine()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	c.process("}")
	return nil
}

func (c *CompilationEngine) compileClassVarDec() error {
	var level string
	var kind string
	if c.curToken() == "static" || c.curToken() == "field" {
		level = c.curToken()
		c.process(c.curToken())
	} else {
		return errors.New("syntax error\n")
	}

	if c.curToken() == "int" || c.curToken() == "boolean" || c.curToken() == "char" {
		kind = c.curToken()
		c.process(c.curToken())
	} else if tokenType(c.curToken()) == IDENTIFIER {
		kind = c.curToken()
		c.process(c.curToken())
	} else {
		return errors.New("syntax error\n")
	}

	c.identifierProcess(c.curToken(), IDENTIFIER, kind, level, "declare")

	for c.curToken() == "," {
		c.process(",")
		c.identifierProcess(c.curToken(), IDENTIFIER, kind, level, "declare")
	}

	c.process(";")

	return nil
}

func (c *CompilationEngine) compileSubroutine() error {
	c.subroutineTable.reset()
	c.subroutineTable.ifIndex = 0
	c.subroutineTable.whileIndex = 0
	c.subroutineTable.functionType = ""
	if c.curToken() == "function" {
		c.process(c.curToken())
		c.vm = append(c.vm, []byte(fmt.Sprintf("function %s.", c.classTable.name))...)
	} else if c.curToken() == "constructor" {
		c.subroutineTable.functionType = "constructor"
		c.process(c.curToken())
		c.vm = append(c.vm, []byte(fmt.Sprintf("function %s.", c.classTable.name))...)
	} else if c.curToken() == "method" {
		c.subroutineTable.functionType = "method"
		c.process(c.curToken())
		// symbol table add "this"
		c.subroutineTable.define("this", c.classTable.name, ARGUMENT)
		c.vm = append(c.vm, []byte(fmt.Sprintf("function %s.", c.classTable.name))...)
	}

	c.subroutineTable.setReturnType(c.curToken())
	if c.curToken() == "void" || c.curToken() == "int" || c.curToken() == "boolean" || c.curToken() == "char" {
		c.process(c.curToken())
	} else if tokenType(c.curToken()) == IDENTIFIER {
		c.process(c.curToken())
	} else {
		return errors.New("syntax error\n")
	}

	c.vm = append(c.vm, []byte(fmt.Sprintf("%s ", c.curToken()))...)
	c.process(c.curToken())
	c.process("(")
	pCount, err := c.compileParameterList()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process(")")

	err = c.compileSubroutineBody(pCount)
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	return nil
}
func (c *CompilationEngine) compileParameterList() (int, error) {
	count := -1
	if c.curToken() != ")" {
		count += 1
		if c.curToken() == "int" || c.curToken() == "boolean" || c.curToken() == "char" {
			c.process(c.curToken())
		} else if tokenType(c.curToken()) == IDENTIFIER {
			c.process(c.curToken())
		} else {
			return 0, errors.New("syntax error\n")
		}

		c.identifierProcess(c.curToken(), tokenType(c.curToken()), c.tokens[c.index-1], ARGUMENT, "declare")

		for c.curToken() == "," {
			count += 1
			c.process(",")
			if c.curToken() == "int" || c.curToken() == "boolean" || c.curToken() == "char" {
				c.process(c.curToken())
			} else if tokenType(c.curToken()) == IDENTIFIER {
				//c.identifierProcess(c.curToken(), tokenType(c.curToken()), c.tokens[c.index-1], ARGUMENT, "declare")
				c.process(c.curToken())
			} else {
				return 0, errors.New("syntax error\n")
			}

			c.identifierProcess(c.curToken(), tokenType(c.curToken()), c.tokens[c.index-1], ARGUMENT, "declare")
		}
	}

	return count + 1, nil
}

func (c *CompilationEngine) compileSubroutineBody(paraCount int) error {
	c.process("{")
	for c.curToken() == "var" {
		err := c.compileVarDec()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}

	// 统计有多少local variable
	count := 0
	for _, v := range c.subroutineTable.table {
		if v.level == LOCAL {
			count += 1
		}
	}
	c.vm = append(c.vm, []byte(fmt.Sprintf("%d\n", count))...)

	if c.subroutineTable.functionType == "constructor" {
		// find local variable number
		fCount := 0
		for _, v := range c.classTable.table {
			if v.level == FIELD {
				fCount += 1
			}
		}
		c.vm = append(c.vm, []byte(fmt.Sprintf("push constant %d\ncall Memory.alloc 1\npop pointer 0\n", fCount))...)
	}
	if c.subroutineTable.functionType == "method" {
		c.vm = append(c.vm, []byte(fmt.Sprintf("push argument 0\npop pointer 0\n"))...)
	}

	err := c.compileStatements()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process("}")
	return nil
}
func (c *CompilationEngine) compileVarDec() error {
	var kind string

	c.process("var")

	if c.curToken() == "int" || c.curToken() == "boolean" || c.curToken() == "char" {
		kind = c.curToken()
		c.process(c.curToken())
	} else if tokenType(c.curToken()) == IDENTIFIER {
		kind = c.curToken()
		c.process(c.curToken())
	} else {
		return errors.New("syntax error\n")
	}

	c.identifierProcess(c.curToken(), tokenType(c.curToken()), kind, LOCAL, "declare")

	for c.curToken() == "," {
		c.process(",")
		c.identifierProcess(c.curToken(), tokenType(c.curToken()), kind, LOCAL, "declare")
	}

	c.process(";")

	return nil
}

func (c *CompilationEngine) compileStatements() error {
	for {
		var err error
		switch c.curToken() {
		case "let":
			err = c.compileLet()
		case "if":
			err = c.compileIf()
		case "while":
			err = c.compileWhile()
		case "do":
			err = c.compileDo()
		case "return":
			err = c.compileReturn()
		default:
			goto outfor
		}
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}

outfor:

	return nil
}

func (c *CompilationEngine) compileLet() error {
	var leftIsArray bool
	c.process("let")
	//var index int
	//var integerType string
	var level string
	var index int
	if c.subroutineTable.indexOf(c.curToken()) > -1 {
		//index = c.subroutineTable.indexOf(c.curToken())
		//integerType = c.subroutineTable.typeOf(c.curToken())
		level = c.subroutineTable.kindOf(c.curToken())
		index = c.subroutineTable.indexOf(c.curToken())
	} else if c.classTable.indexOf(c.curToken()) > -1 {
		//index = c.subroutineTable.indexOf(c.curToken())
		//integerType = c.classTable.typeOf(c.curToken())
		level = c.classTable.kindOf(c.curToken())
		index = c.classTable.indexOf(c.curToken())
	}
	//idx := c.identifierProcess(c.curToken(), tokenType(c.curToken()), integerType, level, "use")
	c.process(c.curToken())

	if c.curToken() == "[" {
		leftIsArray = true
		c.process("[")
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		c.process("]")
		c.vm = append(c.vm, []byte(fmt.Sprintf("push %s %d\nadd\n", levelFunc(level), index))...)
	}
	c.process("=")

	err := c.compileExpression()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	c.process(";")

	if leftIsArray {
		// 把右边expressin的值存入temp0，把左边的地址存入that，再把expression的值入栈，出栈
		c.vm = append(c.vm, []byte(fmt.Sprintf("pop temp 0\npop pointer 1\npush temp 0\npop that 0\n"))...)
	} else {
		c.vm = append(c.vm, []byte(fmt.Sprintf("pop %s %d\n", levelFunc(level), index))...)
	}
	return nil
}

func levelFunc(level string) string {
	level1 := level
	if level == FIELD {
		level1 = "this"
	}
	return level1
}

func (c *CompilationEngine) compileIf() error {
	curIndex := c.subroutineTable.ifIndex
	c.subroutineTable.ifIndex += 1

	c.process("if")
	c.process("(")

	err := c.compileExpression()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	c.process(")")
	c.process("{")

	c.vm = append(c.vm, []byte(fmt.Sprintf("if-goto IF_TRUE%d\ngoto IF_FALSE%d\nlabel IF_TRUE%d\n", curIndex, curIndex, curIndex))...)
	err = c.compileStatements()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process("}")

	// 没有else的时候，没有if end，不影响最终结果
	c.vm = append(c.vm, []byte(fmt.Sprintf("goto IF_END%d\nlabel IF_FALSE%d\n", curIndex, curIndex))...)
	for c.curToken() == "else" {
		c.process("else")
		c.process("{")

		err = c.compileStatements()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		c.process("}")
	}

	c.vm = append(c.vm, []byte(fmt.Sprintf("label IF_END%d\n", curIndex))...)
	return nil
}

func (c *CompilationEngine) compileWhile() error {
	whileIndex := c.subroutineTable.whileIndex
	c.subroutineTable.whileIndex += 1

	c.vm = append(c.vm, []byte(fmt.Sprintf("label WHILE_EXP%d\n", whileIndex))...)
	c.process("while")
	c.process("(")

	err := c.compileExpression()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process(")")
	c.process("{")

	c.vm = append(c.vm, []byte(fmt.Sprintf("not\nif-goto WHILE_END%d\n", whileIndex))...)
	err = c.compileStatements()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process("}")

	c.vm = append(c.vm, []byte(fmt.Sprintf("goto WHILE_EXP%d\nlabel WHILE_END%d\n", whileIndex, whileIndex))...)
	return nil
}

func (c *CompilationEngine) compileDo() error {
	c.process("do")

	if tokenType(c.curToken()) == IDENTIFIER {
		if c.tokens[c.index+1] == "(" {

			err := c.compileTerm()
			if err != nil {
				return err
			}

		} else if c.tokens[c.index+1] == "." {

			// todo 暂时无法和 compileTerm 相同的部分合并，最后的 pop temp 0要根据函数的返回值确定，对于 do 的情况返回值必然是 void，所以pop temp 0，对于其他情况，如何知道返回值是不是 void？如果前一个 token 是 op，那么返回值不是 void
			var beforeDot string
			var integerType string
			var level string
			var index int
			os := 1
			//var varName bool
			if c.subroutineTable.indexOf(c.curToken()) > -1 {
				// varName
				integerType = c.subroutineTable.typeOf(c.curToken())
				level = c.subroutineTable.kindOf(c.curToken())
				index = c.subroutineTable.indexOf(c.curToken())
				beforeDot = integerType
				c.vm = append(c.vm, []byte(fmt.Sprintf("push %s %d\n", levelFunc(level), index))...)
			} else if c.classTable.indexOf(c.curToken()) > -1 {
				// varName
				integerType = c.classTable.typeOf(c.curToken())
				level = c.classTable.kindOf(c.curToken())
				index = c.classTable.indexOf(c.curToken())
				beforeDot = integerType
				c.vm = append(c.vm, []byte(fmt.Sprintf("push %s %d\n", levelFunc(level), index))...)
			} else {
				// className/OS API
				os = 0
				beforeDot = c.curToken()
			}
			// use variable 的地方，可以用process，而不需要identifierProcess
			c.process(c.curToken())
			c.process(".")
			funcName := c.curToken()
			c.process(c.curToken())
			c.process("(")
			nCall, err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")")
			c.vm = append(c.vm, []byte(fmt.Sprintf("call %s.%s %d\npop temp 0\n", beforeDot, funcName, nCall+os))...)
		}
	}
	c.process(";")
	return nil
}
func (c *CompilationEngine) compileReturn() error {
	c.process("return")
	if c.curToken() != ";" {
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	c.process(";")
	if c.subroutineTable.returnType == "void" {
		c.vm = append(c.vm, []byte("push constant 0\n")...)
	}
	c.vm = append(c.vm, []byte("return\n")...)
	return nil
}

func (c *CompilationEngine) compileExpression() error {
	err := c.compileTerm()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	for c.curToken() == "+" || c.curToken() == "-" || c.curToken() == "*" || c.curToken() == "/" || c.curToken() == "&" || c.curToken() == "|" || c.curToken() == "<" || c.curToken() == ">" || c.curToken() == "=" {
		op := c.curToken()
		c.process(c.curToken())
		err = c.compileTerm()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		switch op {
		case "+":
			c.vm = append(c.vm, []byte("add\n")...)
		case "-":
			c.vm = append(c.vm, []byte("sub\n")...)
		case "*":
			c.vm = append(c.vm, []byte("call Math.multiply 2\n")...)
		case "/":
			c.vm = append(c.vm, []byte("call Math.divide 2\n")...)
		case "&":
			c.vm = append(c.vm, []byte("and\n")...)
		case "|":
			c.vm = append(c.vm, []byte("or\n")...)
		case "<":
			c.vm = append(c.vm, []byte("lt\n")...)
		case ">":
			c.vm = append(c.vm, []byte("gt\n")...)
		case "=":
			c.vm = append(c.vm, []byte("eq\n")...)
		}
	}
	return nil
}

func (c *CompilationEngine) compileExpressionList() (int, error) {
	count := -1
	if c.curToken() != ")" {
		count += 1
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return 0, err
		}
		for c.curToken() == "," {
			count += 1
			c.process(",")
			err = c.compileExpression()
			if err != nil {
				fmt.Printf("%v", err)
				return 0, err
			}
		}
	}
	return count + 1, nil
}

func (c *CompilationEngine) compileTerm() error {
	if tokenType(c.curToken()) == INT_CONST {
		c.vm = append(c.vm, []byte(fmt.Sprintf("push constant %s\n", c.curToken()))...)
		c.process(c.curToken())
	} else if tokenType(c.curToken()) == STR_CONST {
		// remove string quote
		str := c.curToken()[1 : len(c.curToken())-1]
		c.vm = append(c.vm, []byte(fmt.Sprintf("push constant %d\ncall String.new 1\n", len(str)))...)
		for _, s := range str {
			c.vm = append(c.vm, []byte(fmt.Sprintf("push constant %d\ncall String.appendChar 2\n", s))...)
		}
		c.process(c.curToken())
	} else if c.curToken() == "true" || c.curToken() == "false" || c.curToken() == "null" || c.curToken() == "this" {
		switch c.curToken() {
		case "true":
			c.vm = append(c.vm, []byte("push constant 0\nnot\n")...)
		case "false", "null":
			c.vm = append(c.vm, []byte("push constant 0\n")...)
		case "this":
			c.vm = append(c.vm, []byte("push pointer 0\n")...)
		}
		c.process(c.curToken())
	} else if tokenType(c.curToken()) == IDENTIFIER {
		// array
		if c.tokens[c.index+1] == "[" {
			var level string
			var index int
			if c.subroutineTable.indexOf(c.curToken()) > -1 {
				level = c.subroutineTable.kindOf(c.curToken())
				index = c.subroutineTable.indexOf(c.curToken())
			} else if c.classTable.indexOf(c.curToken()) > -1 {
				level = c.classTable.kindOf(c.curToken())
				index = c.classTable.indexOf(c.curToken())
			}

			c.process(c.curToken())
			c.process("[")
			err := c.compileExpression()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process("]")
			c.vm = append(c.vm, []byte(fmt.Sprintf("push %s %d\n", levelFunc(level), index))...)
			c.vm = append(c.vm, []byte(fmt.Sprintf("add\npop pointer 1\npush that 0\n"))...)
		} else if c.tokens[c.index+1] == "(" {
			// 相当于 this.function，this 是 pointer 0，也可以去symbol table读取
			c.vm = append(c.vm, []byte(fmt.Sprintf("push pointer 0\n"))...)
			funcName := c.curToken()
			c.process(c.curToken())
			c.process("(")
			// 统计参数数量
			nCall, err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")")
			c.vm = append(c.vm, []byte(fmt.Sprintf("call %s.%s %d\npop temp 0\n", c.classTable.name, funcName, nCall+1))...)

		} else if c.tokens[c.index+1] == "." {
			// subroutine call
			var beforeDot string
			var integerType string
			var level string
			var index int
			// os 为0的时候，OS API/ ClassName.function，不需要var作为第一个argument
			os := 1
			if c.subroutineTable.indexOf(c.curToken()) > -1 {
				integerType = c.subroutineTable.typeOf(c.curToken())
				level = c.subroutineTable.kindOf(c.curToken())
				index = c.subroutineTable.indexOf(c.curToken())
				beforeDot = integerType
			} else if c.classTable.indexOf(c.curToken()) > -1 {
				integerType = c.classTable.typeOf(c.curToken())
				level = c.classTable.kindOf(c.curToken())
				index = c.classTable.indexOf(c.curToken())
				beforeDot = integerType
			} else {
				// OS API/ ClassName.function
				os = 0
				beforeDot = c.curToken()
			}

			c.process(c.curToken())
			c.process(".")
			funcName := c.curToken()
			c.process(c.curToken())
			c.process("(")

			if os == 1 {
				c.vm = append(c.vm, []byte(fmt.Sprintf("push %s %d\n", levelFunc(level), index))...)
			}

			nCall, err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")")

			c.vm = append(c.vm, []byte(fmt.Sprintf("call %s.%s %d\n", beforeDot, funcName, nCall+os))...)
		} else {
			// 只有 identifier
			var level string
			var index int
			if c.subroutineTable.indexOf(c.curToken()) > -1 {
				level = c.subroutineTable.kindOf(c.curToken())
				index = c.subroutineTable.indexOf(c.curToken())
			} else if c.classTable.indexOf(c.curToken()) > -1 {
				level = c.classTable.kindOf(c.curToken())
				index = c.classTable.indexOf(c.curToken())
			}
			// classVarDec 和 VarDec 直接处理了 identifier，不会走到这里，所以也可以用 process
			c.process(c.curToken())
			c.vm = append(c.vm, []byte(fmt.Sprintf("push %s %d\n", levelFunc(level), index))...)
		}
	} else if c.curToken() == "-" || c.curToken() == "~" {
		op := c.curToken()
		c.process(c.curToken())
		err := c.compileTerm()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		switch op {
		case "-":
			c.vm = append(c.vm, []byte("neg\n")...)
		case "~":
			c.vm = append(c.vm, []byte("not\n")...)
		}
	} else if c.curToken() == "(" {
		c.process("(")
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		c.process(")")
	}
	return nil
}

type SymbolTable struct {
	table []SymbolItem
	index map[string]int
}

type ClassSymbolTable struct {
	name string
	SymbolTable
}

func NewClassSymbolTable() *ClassSymbolTable {
	st := SymbolTable{
		table: nil,
		index: map[string]int{
			STATIC: 0,
			FIELD:  0, // this
		},
	}
	return &ClassSymbolTable{SymbolTable: st}
}

func (c *ClassSymbolTable) setName(name string) {
	c.name = name
}

type SubroutineSymbolTable struct {
	ifIndex      int
	whileIndex   int
	returnType   string
	functionType string
	SymbolTable
}

func (c *SubroutineSymbolTable) setReturnType(returnType string) {
	c.returnType = returnType
}

func NewSubroutineSymbolTable() *SubroutineSymbolTable {
	st := SymbolTable{
		table: nil,
		index: map[string]int{
			ARGUMENT: 0,
			LOCAL:    0, // var
		},
	}
	return &SubroutineSymbolTable{SymbolTable: st}
}

type SymbolItem struct {
	name        string
	integerType string // int, boolean, classType etc
	level       string // scopeType static field etc
	index       int
}

func (s *SymbolTable) reset() {
	s.table = nil
	for k := range s.index {
		s.index[k] = 0
	}
}

func (s *SymbolTable) define(name string, integerType string, level string) {
	s.table = append(s.table, SymbolItem{
		name:        name,
		integerType: integerType,
		level:       level,
		index:       s.index[level],
	})
	s.index[level] += 1
}

func (s *SymbolTable) varCount(kind string) int {
	return s.index[kind]
}

func (s *SymbolTable) kindOf(name string) string {
	for _, v := range s.table {
		if v.name == name {
			return v.level
		}
	}
	return NONE
}

func (s *SymbolTable) typeOf(name string) string {
	for _, v := range s.table {
		if v.name == name {
			return v.integerType
		}
	}
	return ""
}

func (s *SymbolTable) indexOf(name string) int {
	for _, v := range s.table {
		if v.name == name {
			return v.index
		}
	}
	return -1
}
