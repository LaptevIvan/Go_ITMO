#include "textflag.h"

// func SumSlice(s []int32) int64
TEXT ·SumSlice(SB), NOSPLIT, $0
    MOVQ pointer+0(FP), AX
    MOVQ n+8(FP), BX
    XORQ CX, CX

loop:
    CMPQ BX, $0
    JEQ end

    MOVLQSX (AX), DX
    ADDQ DX, CX

    ADDQ $4, AX
    SUBQ $1, BX
    JMP loop
end:
    MOVQ CX, ans+24(FP)
    RET

