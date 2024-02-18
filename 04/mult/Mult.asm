// The inputs of this program are the values stored in R0
// and R1 (RAM[0] and RAM[1]). The program computes the product R0 * R1 and stores the result in
// R2 (RAM[2]). Assume that R0 ≥ 0, R1 ≥ 0, and R0 * R1 < 32768 (your program need not test these
// conditions). Your code should not change the values of R0 and R1. The supplied Mult.test script
// and Mult.cmp compare file are designed to test your program on the CPU emulator, using some
// representative R0 and R1 values.

    // i = 0, sum = 0
    @0
    D=A
    @i
    M=D
    @sum
    M=D

(LOOP)
    // if i == RAM[1], jump to end
    @R1
    D=M
    @i
    D=D-M
    @END
    D;JEQ
    // sum = sum + RAM[0]
    @R0
    D=M
    @sum
    M=M+D
    // i = i + 1
    @i
    M=M+1
    @LOOP
    0;JMP

(END)
    @sum
    D=M
    @R2
    M=D
    @END
    0;JMP