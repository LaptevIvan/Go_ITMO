#include "textflag.h"

// func Fibonacci(n uint64) uint64
TEXT ·Fibonacci(SB), NOSPLIT, $0
    MOVQ n+0(FP), AX
    XORQ BX, BX

    CMPQ BX, AX
    CMOVQEQ BX, CX
    JEQ end

    MOVQ $1, CX
    MOVQ $1, DX
loop:
    CMPQ AX, DX
    JEQ end

    MOVQ CX, DI
    ADDQ BX, CX
    MOVQ DI, BX

    ADDQ $1, DX
    JMP loop
end:
    MOVQ CX, ans+8(FP)
    RET

