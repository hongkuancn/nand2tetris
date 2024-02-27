package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	C_ARITHMETIC = "ARITHMETIC"
	C_PUSH       = "PUSH"
	C_POP        = "POP"
	C_GOTO       = "GOTO"
	C_IF         = "IF"
	C_RETURN     = "RETURN"
	C_FUNCTION   = "FUNCTION"
	C_LABEL      = "LABEL"
	C_CALL       = "CALL"
)

// true = -1，false = 0，处理跳转
var loopCnt int

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: assembler <asm file>")
		os.Exit(1)
	}

	asm := os.Args[1]
	fileInfo, err := os.Stat(asm)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	var fileNames []string
	if fileInfo.IsDir() {
		entries, err := os.ReadDir(asm)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name(), ".vm") {
				fileNames = append(fileNames, entry.Name())
			}
			println(entry.Name())
		}
	} else {
		//println(fileInfo.Name())
		fileNames = append(fileNames, fileInfo.Name())
	}
	dir := filepath.Dir(asm)
	println(dir)
	base := filepath.Base(asm)
	println(base)
	names := strings.Split(base, ".")
	//file = names[0]
	loopCnt = 0

	writer := NewCodeWriter()

	converted := make([]byte, 0)

	writer.setFunc("Sys.boot")
	converted = append(converted, writer.writeBootstrap("Sys.init", 0)...)

	//converted = append(converted, []byte("@256\nD=A\n@SP\nM=D\n")...)
	for _, fileName := range fileNames {
		writer.setFile(fileName[:len(fileName)-3])
		parser := NewParser(filepath.Join(dir, fileName))
		err = parser.read()
		if err != nil {
			fmt.Printf("%v", err)
			return
		}

		for _, line := range parser.lines {
			line = strings.TrimSpace(line)
			if len(line) == 0 || line[0:2] == "//" {
				continue
			}
			converted = append(converted, []byte(fmt.Sprintf("// %s\n", line))...)
			// 处理后置的注释
			split := strings.Split(line, "//")
			line = split[0]
			line = strings.TrimSpace(line)

			typ := parser.commandType(line)
			if typ == C_ARITHMETIC {
				converted = append(converted, writer.writeArithmetic(parser.arg1(line))...)
			} else if typ == C_PUSH {
				converted = append(converted, writer.writePushPop(C_PUSH, parser.arg1(line), parser.arg2(line))...)
			} else if typ == C_POP {
				converted = append(converted, writer.writePushPop(C_POP, parser.arg1(line), parser.arg2(line))...)
			} else if typ == C_LABEL {
				converted = append(converted, writer.writeLabel(parser.arg1(line))...)
			} else if typ == C_IF {
				converted = append(converted, writer.writeIf(parser.arg1(line))...)
			} else if typ == C_GOTO {
				converted = append(converted, writer.writeGoto(parser.arg1(line))...)
			} else if typ == C_FUNCTION {
				funcName := parser.arg1(line)
				writer.setFunc(funcName)
				_, ok := writer.retMap[funcName]
				if !ok {
					writer.retMap[funcName] = 0
				}
				converted = append(converted, writer.writeFunction(funcName, parser.arg2(line))...)
			} else if typ == C_RETURN {
				converted = append(converted, writer.writeReturn()...)
			} else if typ == C_CALL {
				converted = append(converted, writer.writeCall(parser.arg1(line), parser.arg2(line))...)
			}
		}
	}

	converted = append(converted, []byte("(END)\n@END\n0;JMP\n")...)

	err = os.WriteFile(filepath.Join(dir, names[0]+".asm"), converted, 0644)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	if len(converted) > 0 {
		fmt.Println("generate symbolic code successfully")
	}
}

type Parser struct {
	file  string
	lines []string
}

func NewParser(file string) *Parser {
	return &Parser{file: file}
}

func (p *Parser) read() error {
	content, err := os.ReadFile(p.file)
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	p.lines = strings.Split(string(content), "\n")
	return nil
}

func (p *Parser) commandType(line string) string {
	split := strings.Split(line, " ")
	switch split[0] {
	case "add", "sub", "neg", "eq", "gt", "lt", "and", "or", "not":
		return C_ARITHMETIC
	case "push":
		return C_PUSH
	case "pop":
		return C_POP
	case "goto":
		return C_GOTO
	case "if-goto":
		return C_IF
	case "call":
		return C_CALL
	case "function":
		return C_FUNCTION
	case "return":
		return C_RETURN
	case "label":
		return C_LABEL
	}
	return ""
}

func (p *Parser) arg1(line string) string {
	split := strings.Split(line, " ")
	if len(split) >= 2 {
		return split[1]
	} else {
		return split[0]
	}
}

func (p *Parser) arg2(line string) int {
	split := strings.Split(line, " ")
	if len(split) >= 3 {
		// 处理数字两边的空格
		tmp := strings.Split(split[2], "\t")
		split[2] = tmp[0]
		atoi, err := strconv.Atoi(split[2])
		if err != nil {
			fmt.Printf("%v", err)
			return -1
		}
		return atoi
	}
	//println("wrong")
	return -1
}

type CodeWriter struct {
	funcs  []string
	file   string
	retMap map[string]int
}

func NewCodeWriter() *CodeWriter {
	return &CodeWriter{retMap: make(map[string]int), funcs: make([]string, 0)}
}

func (c *CodeWriter) setFile(file string) {
	c.file = file
}

func (c *CodeWriter) setFunc(fn string) {
	c.funcs = append(c.funcs, fn)
}

func (c *CodeWriter) returnFunc() {
	c.funcs = c.funcs[:len(c.funcs)-1]
}

// 当前的函数
func (c *CodeWriter) curFunc() string {
	return c.funcs[len(c.funcs)-1]
}

func (c *CodeWriter) writeArithmetic(command string) []byte {
	res := make([]byte, 0)
	println(command)
	if command == "neg" || command == "not" {
		res = append(res, []byte("@SP\nM=M-1\nA=M\n")...)
		switch command {
		case "neg":
			res = append(res, []byte("M=-M\n")...)
		case "not":
			res = append(res, []byte("M=!M\n")...)
		}
	} else {
		res = append(res, []byte("@SP\nM=M-1\nA=M\nD=M\n@SP\nM=M-1\nA=M\n")...)
		switch command {
		case "add":
			res = append(res, []byte("M=M+D\n")...)
		case "sub":
			res = append(res, []byte("M=M-D\n")...)
		case "eq":
			res = append(res, []byte("D=M-D\n@$TRUE$")...)
			res = append(res, []byte(strconv.Itoa(loopCnt))...)
			res = append(res, []byte("\nD;JEQ\n@SP\nA=M\nM=0\n@$END$")...)
			res = append(res, compare()...)
			loopCnt += 1
		case "gt":
			res = append(res, []byte("D=M-D\n@$TRUE$")...)
			res = append(res, []byte(strconv.Itoa(loopCnt))...)
			res = append(res, []byte("\nD;JGT\n@SP\nA=M\nM=0\n@$END$")...)
			res = append(res, compare()...)
			loopCnt += 1
		case "lt":
			res = append(res, []byte("D=M-D\n@$TRUE$")...)
			res = append(res, []byte(strconv.Itoa(loopCnt))...)
			res = append(res, []byte("\nD;JLT\n@SP\nA=M\nM=0\n@$END$")...)
			res = append(res, compare()...)
			loopCnt += 1
		case "and":
			res = append(res, []byte("M=M&D\n")...)
		case "or":
			res = append(res, []byte("M=M|D\n")...)
		}
	}
	res = append(res, []byte("@SP\nM=M+1\n")...)

	return res
}

func compare() []byte {
	res := make([]byte, 0)
	res = append(res, []byte(strconv.Itoa(loopCnt))...)
	res = append(res, []byte("\n0;JMP\n($TRUE$")...)
	res = append(res, []byte(strconv.Itoa(loopCnt))...)
	res = append(res, []byte(")\n@SP\nA=M\nM=-1\n($END$")...)
	res = append(res, []byte(strconv.Itoa(loopCnt))...)
	res = append(res, []byte(")\n")...)
	return res
}

func (c *CodeWriter) writePushPop(command string, seg string, index int) []byte {
	res := make([]byte, 0)
	if command == C_PUSH {
		switch seg {
		case "argument":
			res = append(res, c.pushHelper("ARG", index)...)
		case "local":
			res = append(res, c.pushHelper("LCL", index)...)
		case "static":
			res = append(res, []byte(fmt.Sprintf("@%s.%d\nD=M\n", c.file, index))...)
		case "constant":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n")...)
		case "this":
			res = append(res, c.pushHelper("THIS", index)...)
		case "that":
			res = append(res, c.pushHelper("THAT", index)...)
		case "pointer":
			switch index {
			case 0:
				res = append(res, []byte("@THIS\nD=M\n")...)
			case 1:
				res = append(res, []byte("@THAT\nD=M\n")...)
			}
		case "temp":
			res = append(res, []byte(fmt.Sprintf("@%d\nD=A\n@5\nA=A+D\nD=M\n", index))...)
		}
		res = append(res, []byte("@SP\nA=M\nM=D\n@SP\nM=M+1\n")...)
	} else {
		res = append(res, []byte("@SP\nM=M-1\nA=M\nD=M\n@R13\nM=D\n")...)
		switch seg {
		case "argument":
			res = append(res, c.popHelper("ARG", index)...)
		case "local":
			res = append(res, c.popHelper("LCL", index)...)
		case "static":
			res = append(res, []byte(fmt.Sprintf("@%s.%d\nM=D\n", c.file, index))...)
		case "this":
			res = append(res, c.popHelper("THIS", index)...)
		case "that":
			res = append(res, c.popHelper("THAT", index)...)
		case "pointer":
			switch index {
			case 0:
				res = append(res, []byte("@THIS\nM=D\n")...)
			case 1:
				res = append(res, []byte("@THAT\nM=D\n")...)
			}
		case "temp":
			res = append(res, []byte(fmt.Sprintf("@%d\nD=A\n@5\nD=A+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n", index))...)
		}
	}
	return res
}

func (c *CodeWriter) pushHelper(seg string, index int) []byte {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("@%d\nD=A\n@%s\nA=M+D\nD=M\n", index, seg))
	return []byte(builder.String())
}

func (c *CodeWriter) popHelper(seg string, index int) []byte {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("@%d\nD=A\n@%s\nD=M+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n", index, seg))
	return []byte(builder.String())
}

func (c *CodeWriter) writeLabel(label string) []byte {
	res := make([]byte, 0)
	res = append(res, []byte(fmt.Sprintf("(%s$%s)\n", c.curFunc(), label))...)
	return res
}

func (c *CodeWriter) writeIf(label string) []byte {
	res := make([]byte, 0)
	res = append(res, []byte(fmt.Sprintf("@SP\nM=M-1\nA=M\nD=M\n@%s$%s\nD;JNE\n", c.curFunc(), label))...)
	return res
}

func (c *CodeWriter) writeGoto(label string) []byte {
	res := make([]byte, 0)
	res = append(res, []byte(fmt.Sprintf("@%s$%s\nD;JMP\n", c.curFunc(), label))...)
	return res
}

func (c *CodeWriter) writeFunction(fuc string, nArgs int) []byte {
	ress := fmt.Sprintf("(%s)\n@%d\nD=A\n(%s$push)\n@%s$endpush\nD;JEQ\n@SP\nA=M\nM=0\n@SP\nM=M+1\nD=D-1\n@%s$push\n0;JMP\n(%s$endpush)\n", fuc, nArgs, fuc, fuc, fuc, fuc)
	return []byte(ress)
}

func (c *CodeWriter) writeReturn() []byte {
	builder := strings.Builder{}
	// 必须先保存return address，对于没有argument的函数，return value会覆盖return address，R14先保存return address
	builder.WriteString("@LCL\nD=M\n@R15\nM=D\n// returnAddr=*(frame-5)\n@5\nD=A\n@R15\nA=M-D\nD=M\n@R14\nM=D\n@SP\nM=M-1\nA=M\nD=M\n// *ARG=pop()\n@ARG\nA=M\nM=D\n// SP=ARG+1\n@ARG\nD=M\n@SP\nM=D+1\n")
	builder.WriteString("// pop THAT\n@R15\nAM=M-1\nD=M\n@THAT\nM=D\n")
	builder.WriteString("// pop THIS\n@R15\nAM=M-1\nD=M\n@THIS\nM=D\n")
	builder.WriteString("// pop ARG\n@R15\nAM=M-1\nD=M\n@ARG\nM=D\n")
	builder.WriteString("// pop LCL\n@R15\nAM=M-1\nD=M\n@LCL\nM=D\n")
	builder.WriteString("@R14\nA=M\n0;JMP\n")
	c.returnFunc()
	return []byte(builder.String())
}

func (c *CodeWriter) writeCall(label string, nArgs int) []byte {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("@%s$ret.%d\nD=A\n@SP\nA=M\nM=D\n@SP\nM=M+1\n", c.curFunc(), c.retMap[c.curFunc()]))
	builder.WriteString("// push local\n@LCL\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString("// push arg\n@ARG\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString("// push this\n@THIS\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString("// push that\n@THAT\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString(fmt.Sprintf("// arg = sp-5-args\n@%d\nD=A\n@5\nD=A+D\n@SP\nD=M-D\n@ARG\nM=D\n", nArgs))
	builder.WriteString("// local=sp\n@SP\nD=M\n@LCL\nM=D\n")
	builder.WriteString(fmt.Sprintf("// goto f\n@%s\n0;JMP\n", label))
	builder.WriteString(fmt.Sprintf("(%s$ret.%d)\n", c.curFunc(), c.retMap[c.curFunc()]))
	// ret计数加1
	c.retMap[c.curFunc()] += 1

	return []byte(builder.String())
}

func (c *CodeWriter) writeBootstrap(label string, nArgs int) []byte {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("@256\nD=A\n@SP\nM=D\n"))
	builder.WriteString(fmt.Sprintf("@%s$ret.%d\nD=A\n@SP\nA=M\nM=D\n@SP\nM=M+1\n", c.curFunc(), c.retMap[c.curFunc()]))
	builder.WriteString("// push local\n@LCL\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString("// push arg\n@ARG\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString("// push this\n@THIS\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString("// push that\n@THAT\nD=M\n@SP\nA=M\nM=D\n@SP\nM=M+1\n")
	builder.WriteString(fmt.Sprintf("// arg = sp-5-args\n@%d\nD=A\n@5\nD=A+D\n@SP\nD=M-D\n@ARG\nM=D\n", nArgs))
	builder.WriteString("// local=sp\n@SP\nD=M\n@LCL\nM=D\n")
	builder.WriteString(fmt.Sprintf("// goto f\n@%s\n0;JMP\n", label))
	builder.WriteString(fmt.Sprintf("(%s$ret.%d)\n", c.curFunc(), c.retMap[c.curFunc()]))
	// ret计数加1
	c.retMap[c.curFunc()] += 1

	return []byte(builder.String())
}
