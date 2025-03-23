#include "textflag.h"

// func LowerBound(slice []int64, value int64) int64
TEXT ·LowerBound(SB), NOSPLIT, $0
    MOVQ pointer+0(FP), AX
    MOVQ value+24(FP), BX

    MOVQ $-1, CX
    MOVQ n+8(FP), DX

loop:
    MOVQ CX, DI
    ADDQ $1, DI
    CMPQ DX, DI
    JEQ end

    MOVQ DX, DI
    SUBQ CX, DI
    SHRQ $1, DI
    ADDQ CX, DI
    MOVQ (AX)(DI*8), R8

    CMPQ R8, BX
    JLE less
    JMP more

less:
    MOVQ DI, CX
    JMP loop

more:
    MOVQ DI, DX
    JMP loop
end:
    MOVQ CX, ans+32(FP)
    RET

