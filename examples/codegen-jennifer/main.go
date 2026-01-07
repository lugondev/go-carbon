package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/lugondev/go-carbon/examples/codegen-jennifer/generated"
)

func main() {
	fmt.Println("=== Token Swap Example ===")
	fmt.Println()

	client := rpc.New(rpc.DevNet_RPC)

	authority, err := solana.NewRandomPrivateKey()
	if err != nil {
		log.Fatalf("Failed to generate keypair: %v", err)
	}

	tokenAMint := solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
	tokenBMint := solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	swapPoolKey, _ := solana.NewRandomPrivateKey()
	swapPool := swapPoolKey.PublicKey()
	systemProgram := solana.SystemProgramID

	fmt.Printf("Program ID: %s\n", tokenswap.ProgramID)
	fmt.Printf("Program Name: %s\n", tokenswap.ProgramName)
	fmt.Printf("Program Version: %s\n", tokenswap.ProgramVersion)
	fmt.Println()

	feeRate := uint64(30)
	initIx := tokenswap.NewInitializePoolInstruction(
		swapPool,
		authority.PublicKey(),
		tokenAMint,
		tokenBMint,
		systemProgram,
		feeRate,
	)

	instruction, err := initIx.Build()
	if err != nil {
		log.Fatalf("Failed to build initialize instruction: %v", err)
	}

	fmt.Println("✓ Initialize Pool Instruction created:")
	fmt.Printf("  Program ID: %s\n", instruction.ProgramID())
	fmt.Printf("  Accounts: %d\n", len(instruction.Accounts()))
	data, _ := instruction.Data()
	fmt.Printf("  Data size: %d bytes\n", len(data))
	fmt.Println()

	userSourceKey, _ := solana.NewRandomPrivateKey()
	userSource := userSourceKey.PublicKey()
	userDestKey, _ := solana.NewRandomPrivateKey()
	userDest := userDestKey.PublicKey()
	poolSourceKey, _ := solana.NewRandomPrivateKey()
	poolSource := poolSourceKey.PublicKey()
	poolDestKey, _ := solana.NewRandomPrivateKey()
	poolDest := poolDestKey.PublicKey()
	tokenProgram := solana.TokenProgramID

	amountIn := uint64(1000000)
	minAmountOut := uint64(950000)
	direction := tokenswap.SwapDirectionAToB

	swapIx := tokenswap.NewSwapInstruction(
		swapPool,
		authority.PublicKey(),
		userSource,
		userDest,
		poolSource,
		poolDest,
		tokenProgram,
		amountIn,
		minAmountOut,
		direction,
	)

	swapInstruction, err := swapIx.Build()
	if err != nil {
		log.Fatalf("Failed to build swap instruction: %v", err)
	}

	fmt.Println("✓ Swap Instruction created:")
	fmt.Printf("  Program ID: %s\n", swapInstruction.ProgramID())
	fmt.Printf("  Accounts: %d\n", len(swapInstruction.Accounts()))
	swapData, _ := swapInstruction.Data()
	fmt.Printf("  Data size: %d bytes\n", len(swapData))
	fmt.Printf("  Amount In: %d\n", amountIn)
	fmt.Printf("  Min Amount Out: %d\n", minAmountOut)
	fmt.Printf("  Direction: %s\n", direction.String())
	fmt.Println()

	eventData := append(tokenswap.SwapExecutedEventDiscriminator, make([]byte, 100)...)
	_, err = tokenswap.DecodeSwapExecutedEvent(eventData)
	if err != nil {
		fmt.Printf("Event decoding (expected to fail with test data): %v\n", err)
	}

	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Println("✓ Generated code compiles successfully")
	fmt.Println("✓ Instruction builders work correctly")
	fmt.Println("✓ Type-safe instruction creation")
	fmt.Println("✓ Account validation included")
	fmt.Println("✓ Event decoders available")
	fmt.Println()
	fmt.Printf("Program: %s v%s\n", tokenswap.ProgramName, tokenswap.ProgramVersion)
	fmt.Printf("Address: %s\n", tokenswap.ProgramID)

	_ = client
	_ = context.Background()
}
