package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const (
	A_INSTRUCTION = "A"
	C_INSTRUCTION = "C"
	L_INSTRUCTION = "L"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: assembler <asm file>")
		os.Exit(1)
	}
	symTable := NewSymbolTable()
	symTable.init()
	coder := NewCoder()
	parser := NewParser(os.Args[1])
	err := parser.read()
	if err != nil {
		return
	}

	lineCnt := 0
	for _, line := range parser.lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0:2] == "//" {
			continue
		}

		typ := parser.instructionType(line)
		if typ == A_INSTRUCTION || typ == C_INSTRUCTION {
			lineCnt += 1
		} else if typ == L_INSTRUCTION {
			symTable.addEntry(parser.symbol(line), lineCnt)
		}
	}

	converted := make([]byte, 0)
	nextVar := 16
	for _, line := range parser.lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0:2] == "//" {
			continue
		}

		typ := parser.instructionType(line)
		if typ == A_INSTRUCTION {
			var decimal int
			sym := parser.symbol(line)
			if unicode.IsDigit(rune(sym[0])) {
				decimal, _ = strconv.Atoi(sym)
			} else {
				decimal = symTable.getAddress(sym)
				if decimal == -1 {
					decimal = nextVar
					symTable.addEntry(sym, nextVar)
					nextVar += 1
				}
			}
			binary := fmt.Sprintf("%016b\n", decimal)
			converted = append(converted, []byte(binary)...)
		} else if typ == C_INSTRUCTION {
			converted = append(converted, []byte("111")...)
			converted = append(converted, []byte(coder.comp(parser.comp(line)))...)
			converted = append(converted, []byte(coder.dest(parser.dest(line)))...)
			converted = append(converted, []byte(coder.jump(parser.jump(line)))...)
			converted = append(converted, []byte("\n")...)
		}
	}

	asm := os.Args[1]
	dir := filepath.Dir(asm)
	base := filepath.Base(asm)
	names := strings.Split(base, ".")
	err = os.WriteFile(filepath.Join(dir, names[0]+".hack"), converted, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(converted) > 0 {
		fmt.Println("generate hack code successfully")
	}
}

type Parser struct {
	file  string
	lines []string
}

func NewParser(input string) *Parser {
	return &Parser{file: input}
}

func (p *Parser) read() error {
	content, err := os.ReadFile(p.file)
	if err != nil {
		return err
	}
	p.lines = strings.Split(string(content), "\n")
	return nil
}

func (p *Parser) instructionType(line string) string {
	line = strings.TrimSpace(line)
	if line[0] == '@' {
		return A_INSTRUCTION
	}
	if line[0] == '(' {
		return L_INSTRUCTION
	}
	return C_INSTRUCTION
}

func (p *Parser) symbol(line string) string {
	typ := p.instructionType(line)
	if typ == A_INSTRUCTION {
		return line[1:]
	} else if typ == L_INSTRUCTION {
		return line[1 : len(line)-1]
	}

	return ""
}

func (p *Parser) dest(line string) string {
	equalIdx := strings.Index(line, "=")
	if equalIdx >= 0 {
		return line[:equalIdx]
	}
	return ""
}

func (p *Parser) comp(line string) string {
	equalIdx := strings.Index(line, "=")
	semiIdx := strings.Index(line, ";")
	if semiIdx < 0 {
		if equalIdx < 0 {
			return line
		} else {
			return line[equalIdx+1:]
		}
	} else {
		if equalIdx < 0 {
			return line[:semiIdx]
		} else {
			return line[equalIdx+1 : semiIdx]
		}
	}
}

func (p *Parser) jump(line string) string {
	semiIdx := strings.Index(line, ";")
	if semiIdx >= 0 {
		return line[semiIdx+1:]
	}
	return ""
}

type Coder struct {
}

func NewCoder() *Coder {
	return &Coder{}
}

func (c *Coder) dest(part string) string {
	switch part {
	case "M":
		return "001"
	case "D":
		return "010"
	case "DM":
		fallthrough
	case "MD":
		return "011"
	case "A":
		return "100"
	case "AM":
		return "101"
	case "AD":
		return "110"
	case "ADM":
		fallthrough
	case "AMD":
		return "111"
	default:
		return "000"
	}
}

func (c *Coder) comp(part string) string {
	switch part {
	case "0":
		return "0101010"
	case "1":
		return "0111111"
	case "-1":
		return "0111010"
	case "D":
		return "0001100"
	case "A":
		return "0110000"
	case "M":
		return "1110000"
	case "!D":
		return "0001101"
	case "!A":
		return "0110001"
	case "!M":
		return "1110001"
	case "-D":
		return "0001111"
	case "-A":
		return "0110011"
	case "-M":
		return "1110011"
	case "D+1":
		return "0011111"
	case "A+1":
		return "0110111"
	case "M+1":
		return "1110111"
	case "D-1":
		return "0001110"
	case "A-1":
		return "0110010"
	case "M-1":
		return "1110010"
	case "D+A":
		return "0000010"
	case "D+M":
		return "1000010"
	case "D-A":
		return "0010011"
	case "D-M":
		return "1010011"
	case "A-D":
		return "0000111"
	case "M-D":
		return "1000111"
	case "D&A":
		return "0000000"
	case "D&M":
		return "1000000"
	case "D|A":
		return "0010101"
	case "D|M":
		return "1010101"
	default:
		return "0000000"
	}
}

func (c *Coder) jump(part string) string {
	switch part {
	case "JGT":
		return "001"
	case "JEQ":
		return "010"
	case "JGE":
		return "011"
	case "JLT":
		return "100"
	case "JNE":
		return "101"
	case "JLE":
		return "110"
	case "JMP":
		return "111"
	default:
		return "000"
	}
}

type SymbolTable struct {
	st      map[string]int
	nextVar int
}

func NewSymbolTable() *SymbolTable {
	st := make(map[string]int)
	return &SymbolTable{st: st}
}

func (s *SymbolTable) init() {
	s.addEntry("R0", 0)
	s.addEntry("R1", 1)
	s.addEntry("R2", 2)
	s.addEntry("R3", 3)
	s.addEntry("R4", 4)
	s.addEntry("R5", 5)
	s.addEntry("R6", 6)
	s.addEntry("R7", 7)
	s.addEntry("R8", 8)
	s.addEntry("R9", 9)
	s.addEntry("R10", 10)
	s.addEntry("R11", 11)
	s.addEntry("R12", 12)
	s.addEntry("R13", 13)
	s.addEntry("R14", 14)
	s.addEntry("R15", 15)
	s.addEntry("SP", 0)
	s.addEntry("LCL", 1)
	s.addEntry("ARG", 2)
	s.addEntry("THIS", 3)
	s.addEntry("THAT", 4)
	s.addEntry("SCREEN", 16384)
	s.addEntry("KBD", 24576)
}

func (s *SymbolTable) addEntry(symbol string, address int) {
	s.st[symbol] = address
}

func (s *SymbolTable) contains(symbol string) bool {
	_, ok := s.st[symbol]
	return ok
}

func (s *SymbolTable) getAddress(symbol string) int {
	address, ok := s.st[symbol]
	if !ok {
		return -1
	}
	return address
}
