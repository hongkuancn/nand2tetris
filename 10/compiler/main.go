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
	SYMBOL     = "symbol"
	KEYWORD    = "keyword"
	IDENTIFIER = "identifier"
	INT_CONST  = "int_const"
	STR_CONST  = "str_const"
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
		os.Exit(1)
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

	var converted []byte
	// generate token file
	for _, fileName := range fileNames {
		tokenizer := NewJackTokenizer(filepath.Join(dir, fileName))
		err := tokenizer.read()
		if err != nil {
			return
		}
		engine.setTokens(tokenizer.tokens)
		converted = nil
		converted = append(converted, []byte("<tokens>\n")...)

		for i := 0; i < len(tokenizer.tokens); i++ {
			t := tokenizer.tokenType(tokenizer.tokens[i])
			switch t {
			case INT_CONST:
				converted = append(converted, []byte(fmt.Sprintf("<intConst> %s </intConst>\n", tokenizer.tokens[i]))...)
			case STR_CONST:
				converted = append(converted, []byte(fmt.Sprintf("<stringConst> %s </stringConst>\n", tokenizer.tokens[i]))...)
			case SYMBOL:
				switch tokenizer.tokens[i] {
				case "<":
					converted = append(converted, []byte(fmt.Sprintf("<symbol> %s </symbol>\n", "&lt;"))...)
				case ">":
					converted = append(converted, []byte(fmt.Sprintf("<symbol> %s </symbol>\n", "&gt;"))...)
				case "\"":
					converted = append(converted, []byte(fmt.Sprintf("<symbol> %s </symbol>\n", "&quot;"))...)
				case "&":
					converted = append(converted, []byte(fmt.Sprintf("<symbol> %s </symbol>\n", "&amp;"))...)
				default:
					converted = append(converted, []byte(fmt.Sprintf("<symbol> %s </symbol>\n", tokenizer.tokens[i]))...)
				}
			case IDENTIFIER:
				converted = append(converted, []byte(fmt.Sprintf("<identifier> %s </identifier>\n", tokenizer.tokens[i]))...)
			case KEYWORD:
				converted = append(converted, []byte(fmt.Sprintf("<keyword> %s </keyword>\n", tokenizer.tokens[i]))...)
			}
		}

		converted = append(converted, []byte("</tokens>\n")...)

		names := strings.Split(fileName, ".")

		fmt.Printf("%s\n", names[0])
		err = os.WriteFile(filepath.Join(dir, names[0]+"1T.xml"), converted, 0644)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		if len(converted) > 0 {
			fmt.Println("generate vm token file successfully")
		}

		engine.index = 0
		engine.res = nil
		err = engine.compileClass()
		if err != nil {
			return
		}

		err = os.WriteFile(filepath.Join(dir, names[0]+"1.xml"), engine.res, 0644)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		if len(engine.res) > 0 {
			fmt.Println("generate parse tree xml file successfully")
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
	index  int
	res    []byte
	tokens []string
}

func NewCompilationEngine() *CompilationEngine {
	return &CompilationEngine{}
}

func (c *CompilationEngine) process(expected string, typ string) {
	if c.tokens[c.index] == expected {
		c.res = append(c.res, []byte(fmt.Sprintf("<%s> %s </%s>\n", typ, expected, typ))...)
	} else {
		//fmt.Println(string(c.res))
		//fmt.Printf("expected %s real %s\n", expected, c.tokens[c.index])
		// for simplicity
		os.Exit(1)
	}
	c.index += 1
}

func (c *CompilationEngine) setTokens(tokens []string) {
	c.tokens = tokens
}

func (c *CompilationEngine) compileClass() error {
	c.res = append(c.res, []byte("<class>\n")...)
	c.process("class", KEYWORD)
	c.process(c.tokens[c.index], IDENTIFIER)
	c.process("{", SYMBOL)
	for c.tokens[c.index] == "static" || c.tokens[c.index] == "field" {
		err := c.compileClassVarDec()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	for c.tokens[c.index] == "constructor" || c.tokens[c.index] == "function" || c.tokens[c.index] == "method" {
		err := c.compileSubroutine()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	c.process("}", SYMBOL)
	c.res = append(c.res, []byte("</class>\n")...)
	return nil
}

func (c *CompilationEngine) compileClassVarDec() error {
	c.res = append(c.res, []byte("<classVarDec>\n")...)

	if c.tokens[c.index] == "static" || c.tokens[c.index] == "field" {
		c.process(c.tokens[c.index], KEYWORD)
	} else {
		fmt.Printf("%s\n", string(c.res))
		return errors.New("syntax error\n")
	}

	if c.tokens[c.index] == "int" || c.tokens[c.index] == "boolean" || c.tokens[c.index] == "char" {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else if tokenType(c.tokens[c.index]) == IDENTIFIER {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else {
		fmt.Printf("%s\n", string(c.res))
		return errors.New("syntax error\n")
	}

	c.process(c.tokens[c.index], IDENTIFIER)

	for c.tokens[c.index] == "," {
		c.process(",", SYMBOL)
		c.process(c.tokens[c.index], IDENTIFIER)
	}

	c.process(";", SYMBOL)

	c.res = append(c.res, []byte("</classVarDec>\n")...)

	return nil
}

func (c *CompilationEngine) compileSubroutine() error {
	c.res = append(c.res, []byte("<subroutineDec>\n")...)
	if c.tokens[c.index] == "constructor" || c.tokens[c.index] == "function" || c.tokens[c.index] == "method" {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else {
		fmt.Printf("%s\n", string(c.res))
		return errors.New("syntax error\n")
	}

	if c.tokens[c.index] == "void" || c.tokens[c.index] == "int" || c.tokens[c.index] == "boolean" || c.tokens[c.index] == "char" {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else if tokenType(c.tokens[c.index]) == IDENTIFIER {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else {
		fmt.Printf("%s\n", string(c.res))
		return errors.New("syntax error\n")
	}

	c.process(c.tokens[c.index], IDENTIFIER)
	c.process("(", tokenType("("))
	err := c.compileParameterList()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process(")", tokenType(")"))

	err = c.compileSubroutineBody()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	c.res = append(c.res, []byte("</subroutineDec>\n")...)
	return nil
}
func (c *CompilationEngine) compileParameterList() error {
	c.res = append(c.res, []byte("<parameterList>\n")...)
	if c.tokens[c.index] != ")" {
		if c.tokens[c.index] == "void" || c.tokens[c.index] == "int" || c.tokens[c.index] == "boolean" || c.tokens[c.index] == "char" {
			c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
		} else if tokenType(c.tokens[c.index]) == IDENTIFIER {
			c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
		} else {
			fmt.Printf("%s\n", string(c.res))
			return errors.New("syntax error\n")
		}

		c.process(c.tokens[c.index], IDENTIFIER)

		for c.tokens[c.index] == "," {
			c.process(",", SYMBOL)
			if c.tokens[c.index] == "void" || c.tokens[c.index] == "int" || c.tokens[c.index] == "boolean" || c.tokens[c.index] == "char" {
				c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
			} else if tokenType(c.tokens[c.index]) == IDENTIFIER {
				c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
			} else {
				fmt.Printf("%s\n", string(c.res))
				return errors.New("syntax error\n")
			}

			c.process(c.tokens[c.index], IDENTIFIER)
		}
	}

	c.res = append(c.res, []byte("</parameterList>\n")...)
	return nil
}

func (c *CompilationEngine) compileSubroutineBody() error {
	c.res = append(c.res, []byte("<subroutineBody>\n")...)
	c.process("{", tokenType("{"))
	for c.tokens[c.index] == "var" {
		err := c.compileVarDec()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	err := c.compileStatements()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process("}", tokenType("}"))
	c.res = append(c.res, []byte("</subroutineBody>\n")...)
	return nil
}
func (c *CompilationEngine) compileVarDec() error {

	c.res = append(c.res, []byte("<varDec>\n")...)
	c.process("var", tokenType("var"))

	if c.tokens[c.index] == "int" || c.tokens[c.index] == "boolean" || c.tokens[c.index] == "char" {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else if tokenType(c.tokens[c.index]) == IDENTIFIER {
		c.process(c.tokens[c.index], tokenType(c.tokens[c.index]))
	} else {
		fmt.Printf("%s\n", string(c.res))
		return errors.New("syntax error\n")
	}

	c.process(c.tokens[c.index], IDENTIFIER)

	for c.tokens[c.index] == "," {
		c.process(",", SYMBOL)
		c.process(c.tokens[c.index], IDENTIFIER)
	}

	c.process(";", SYMBOL)
	c.res = append(c.res, []byte("</varDec>\n")...)
	return nil
}

func (c *CompilationEngine) compileStatements() error {
	c.res = append(c.res, []byte("<statements>\n")...)
	for {
		var err error
		switch c.tokens[c.index] {
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
	c.res = append(c.res, []byte("</statements>\n")...)

	return nil
}

func (c *CompilationEngine) compileLet() error {
	c.res = append(c.res, []byte("<letStatements>\n")...)
	c.process("let", KEYWORD)
	c.process(c.tokens[c.index], IDENTIFIER)
	if c.tokens[c.index] == "[" {
		c.process("[", SYMBOL)
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		c.process("]", SYMBOL)
	}
	c.process("=", SYMBOL)
	err := c.compileExpression()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process(";", SYMBOL)
	c.res = append(c.res, []byte("</letStatements>\n")...)
	return nil
}

func (c *CompilationEngine) compileIf() error {
	c.res = append(c.res, []byte("<ifStatements>\n")...)
	c.process("if", KEYWORD)
	c.process("(", SYMBOL)
	err := c.compileExpression()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process(")", SYMBOL)
	c.process("{", SYMBOL)
	err = c.compileStatements()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process("}", SYMBOL)
	for c.tokens[c.index] == "else" {
		c.process("else", KEYWORD)
		c.process("{", SYMBOL)
		err = c.compileStatements()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		c.process("}", SYMBOL)
	}
	c.res = append(c.res, []byte("</ifStatements>\n")...)
	return nil
}

func (c *CompilationEngine) compileWhile() error {
	c.res = append(c.res, []byte("<whileStatements>\n")...)
	c.process("while", KEYWORD)
	c.process("(", SYMBOL)
	err := c.compileExpression()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process(")", SYMBOL)
	c.process("{", SYMBOL)
	err = c.compileStatements()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	c.process("}", SYMBOL)
	c.res = append(c.res, []byte("</whileStatements>\n")...)
	return nil
}

func (c *CompilationEngine) compileDo() error {
	c.res = append(c.res, []byte("<doStatements>\n")...)
	c.process("do", KEYWORD)
	if tokenType(c.tokens[c.index]) == IDENTIFIER {
		// copied from compileTerm
		if c.tokens[c.index+1] == "(" {
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process("(", SYMBOL)
			err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")", SYMBOL)
		} else if c.tokens[c.index+1] == "." {
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process(".", SYMBOL)
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process("(", SYMBOL)
			err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")", SYMBOL)
		}
	}
	c.process(";", SYMBOL)
	c.res = append(c.res, []byte("</doStatements>\n")...)
	return nil
}
func (c *CompilationEngine) compileReturn() error {
	c.res = append(c.res, []byte("<returnStatements>\n")...)
	c.process("return", KEYWORD)
	if c.tokens[c.index] != ";" {
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	c.process(";", SYMBOL)
	c.res = append(c.res, []byte("</returnStatements>\n")...)
	return nil
}

func (c *CompilationEngine) compileExpression() error {
	c.res = append(c.res, []byte("<expression>\n")...)
	err := c.compileTerm()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	for c.tokens[c.index] == "+" || c.tokens[c.index] == "-" || c.tokens[c.index] == "*" || c.tokens[c.index] == "/" || c.tokens[c.index] == "&" || c.tokens[c.index] == "|" || c.tokens[c.index] == "<" || c.tokens[c.index] == ">" || c.tokens[c.index] == "=" {
		c.process(c.tokens[c.index], SYMBOL)
		err = c.compileTerm()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	}
	c.res = append(c.res, []byte("</expression>\n")...)
	return nil
}

func (c *CompilationEngine) compileExpressionList() error {
	c.res = append(c.res, []byte("<expressionList>\n")...)
	if c.tokens[c.index] != ")" {
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		for c.tokens[c.index] == "," {
			c.process(",", SYMBOL)
			err = c.compileExpression()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
		}
	}
	c.res = append(c.res, []byte("</expressionList>\n")...)
	return nil
}

func (c *CompilationEngine) compileTerm() error {
	c.res = append(c.res, []byte("<term>\n")...)
	if tokenType(c.tokens[c.index]) == INT_CONST {
		c.process(c.tokens[c.index], INT_CONST)
	} else if tokenType(c.tokens[c.index]) == STR_CONST {
		c.process(c.tokens[c.index], INT_CONST)
	} else if c.tokens[c.index] == "true" || c.tokens[c.index] == "false" || c.tokens[c.index] == "null" || c.tokens[c.index] == "this" {
		c.process(c.tokens[c.index], KEYWORD)
	} else if tokenType(c.tokens[c.index]) == IDENTIFIER {
		if c.tokens[c.index+1] == "[" {
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process("[", SYMBOL)
			err := c.compileExpression()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process("]", SYMBOL)
		} else if c.tokens[c.index+1] == "(" {
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process("(", SYMBOL)
			err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")", SYMBOL)
		} else if c.tokens[c.index+1] == "." {
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process(".", SYMBOL)
			c.process(c.tokens[c.index], IDENTIFIER)
			c.process("(", SYMBOL)
			err := c.compileExpressionList()
			if err != nil {
				fmt.Printf("%v", err)
				return err
			}
			c.process(")", SYMBOL)
		} else {
			c.process(c.tokens[c.index], IDENTIFIER)
		}
	} else if c.tokens[c.index] == "-" || c.tokens[c.index] == "~" {
		c.process(c.tokens[c.index], SYMBOL)
		err := c.compileTerm()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
	} else if c.tokens[c.index] == "(" {
		c.process("(", SYMBOL)
		err := c.compileExpression()
		if err != nil {
			fmt.Printf("%v", err)
			return err
		}
		c.process(")", SYMBOL)
	}
	c.res = append(c.res, []byte("</term>\n")...)
	return nil
}
