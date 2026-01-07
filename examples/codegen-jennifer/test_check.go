package main

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
)

func main() {
	fmt.Printf("SystemProgramID: %s\n", solana.SystemProgramID)
	fmt.Printf("IsZero: %v\n", solana.SystemProgramID.IsZero())
}
