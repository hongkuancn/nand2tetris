// This file is part of www.nand2tetris.org
// and the book "The Elements of Computing Systems"
// by Nisan and Schocken, MIT Press.
// File name: projects/04/Fill.asm

// Runs an infinite loop that listens to the keyboard input.
// When a key is pressed (any key), the program blackens the screen
// by writing 'black' in every pixel;
// the screen should remain fully black as long as the key is pressed. 
// When no key is pressed, the program clears the screen by writing
// 'white' in every pixel;
// the screen should remain fully clear as long as no key is pressed.

    // i = screen, stop = screen + 8192
    @8192
    D=A
    @SCREEN
    D=D+A
    @stop
    M=D
    @SCREEN
    D=A
    @i
    M=D

(LOOP)
    @KBD
    D=M
    @CLEAR
    D;JEQ

(CHANGESCREEN)
    @stop
    D=M
    @i
    D=D-M
    @ENDCHANGESCREEN
    D;JEQ
    // blacken
    @i
    D=M
    A=D
    M=-1
    @i
    M=M+1
    @CHANGESCREEN
    0;JMP

(CLEAR)
    @stop
    D=M
    @i
    D=D-M
    @ENDCHANGESCREEN
    D;JEQ
    // whiten
    @i
    D=M
    A=D
    M=0
    @i
    M=M+1
    @CLEAR
    0;JMP

(ENDCHANGESCREEN)
    // reset screen starting point
    @SCREEN
    D=A
    @i
    M=D
    @LOOP
    0;JMP