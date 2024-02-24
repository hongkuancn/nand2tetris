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
)

var file string
var loopCnt int

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: assembler <asm file>")
		os.Exit(1)
	}

	asm := os.Args[1]
	dir := filepath.Dir(asm)
	base := filepath.Base(asm)
	names := strings.Split(base, ".")
	file = names[0]
	loopCnt = 0

	writer := NewCodeWriter()
	parser := NewParser(os.Args[1])
	err := parser.read()
	if err != nil {
		return
	}

	converted := make([]byte, 0)

	//converted = append(converted, []byte("@256\nD=A\n@SP\nM=D\n")...)

	for _, line := range parser.lines {

		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0:2] == "//" {
			continue
		}
		converted = append(converted, []byte("// ")...)
		converted = append(converted, []byte(line)...)
		converted = append(converted, []byte("\n")...)
		typ := parser.commandType(line)
		if typ == C_ARITHMETIC {
			converted = append(converted, writer.writeArithmetic(parser.arg1(line))...)
		} else if typ == C_PUSH {
			converted = append(converted, writer.writePushPop(C_PUSH, parser.arg1(line), parser.arg2(line))...)
		} else if typ == C_POP {
			converted = append(converted, writer.writePushPop(C_POP, parser.arg1(line), parser.arg2(line))...)
		}
	}

	converted = append(converted, []byte("(END)\n@END\n0;JMP\n")...)

	err = os.WriteFile(filepath.Join(dir, names[0]+".asm"), converted, 0644)
	if err != nil {
		fmt.Println(err)
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
	}
	return ""
}

func (p *Parser) arg1(line string) string {
	split := strings.Split(line, " ")
	if len(split) == 3 {
		return split[1]
	} else {
		return split[0]
	}
}

func (p *Parser) arg2(line string) int {
	split := strings.Split(line, " ")
	if len(split) == 3 {
		atoi, err := strconv.Atoi(split[2])
		if err != nil {
			return -1
		}
		return atoi
	}
	return -1
}

type CodeWriter struct {
}

func NewCodeWriter() *CodeWriter {
	return &CodeWriter{}
}

func (c *CodeWriter) writeArithmetic(command string) []byte {
	res := make([]byte, 0)
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
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@ARG\nA=M+D\nD=M\n")...)
		case "local":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@LCL\nA=M+D\nD=M\n")...)
		case "static":
			res = append(res, []byte("@")...)
			res = append(res, []byte(file)...)
			res = append(res, []byte(".")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=M\n")...)
		case "constant":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n")...)
		case "this":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@THIS\nA=M+D\nD=M\n")...)
		case "that":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@THAT\nA=M+D\nD=M\n")...)
		case "pointer":
			switch index {
			case 0:
				res = append(res, []byte("@THIS\nD=M\n")...)
			case 1:
				res = append(res, []byte("@THAT\nD=M\n")...)
			}
		case "temp":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@5\nA=A+D\nD=M\n")...)
		}
		res = append(res, []byte("@SP\nA=M\nM=D\n@SP\nM=M+1\n")...)
	} else {
		res = append(res, []byte("@SP\nM=M-1\nA=M\nD=M\n@R13\nM=D\n")...)
		switch seg {
		case "argument":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@ARG\nD=M+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n")...)
		case "local":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@LCL\nD=M+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n")...)
		case "static":
			res = append(res, []byte("@")...)
			res = append(res, []byte(file)...)
			res = append(res, []byte(".")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nM=D\n")...)
		case "this":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@THIS\nD=M+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n")...)
		case "that":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@THAT\nD=M+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n")...)
		case "pointer":
			switch index {
			case 0:
				res = append(res, []byte("@THIS\nM=D\n")...)
			case 1:
				res = append(res, []byte("@THAT\nM=D\n")...)
			}
		case "temp":
			res = append(res, []byte("@")...)
			res = append(res, []byte(strconv.Itoa(index))...)
			res = append(res, []byte("\nD=A\n@5\nD=A+D\n@R14\nM=D\n@R13\nD=M\n@R14\nA=M\nM=D\n")...)
		}

	}
	return res
}
