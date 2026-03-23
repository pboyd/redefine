#include "go_asm.h"
#include "textflag.h"

// getFrame returns the value of the frame pointer register. getFrame doesn't
// have a stack frame of its own so this will be the caller's stack frame.
TEXT ·getFrame(SB),NOSPLIT,$0-8
    MOVD R29, ret+8(SP)
    RET

TEXT ·getg(SB),NOSPLIT,$0-8
    MOVD g, ret+8(SP)	// g is an alias for x28
    RET

TEXT ·dupMarker(SB),NOSPLIT,$0-8
    ADR marker, R0
    MOVD R0, ret+8(SP)
    RET
marker:
    WORD $0
