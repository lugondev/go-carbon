# Jennifer Quick Reference - Code Generation Patterns

## Basic Patterns

### 1. Function Declaration

```go
// func NewSwap(amount uint64) error
Func().Id("NewSwap").
    Params(Id("amount").Uint64()).
    Params(Error()).
    Block(
        Return(Nil()),
    )
```

### 2. Method Declaration

```go
// func (s *Swap) Execute() error
Func().
    Params(Id("s").Op("*").Id("Swap")).
    Id("Execute").
    Params().
    Params(Error()).
    Block(
        Return(Nil()),
    )
```

### 3. Struct Declaration

```go
// type Swap struct {
//     Amount uint64 `json:"amount"`
//     User   solana.PublicKey `json:"user"`
// }
Type().Id("Swap").Struct(
    Id("Amount").Uint64().Tag(map[string]string{"json": "amount"}),
    Id("User").Qual("github.com/gagliardetto/solana-go", "PublicKey").Tag(map[string]string{"json": "user"}),
)
```

### 4. Interface Declaration

```go
// type Swapper interface {
//     Swap(amount uint64) error
// }
Type().Id("Swapper").Interface(
    Id("Swap").Params(Id("amount").Uint64()).Params(Error()),
)
```

### 5. Variable Declaration

```go
// var discriminator = [8]byte{0x01, 0x02, 0x03}
Var().Id("discriminator").Op("=").Index(Lit(8)).Byte().Values(
    Lit(0x01), Lit(0x02), Lit(0x03),
)
```

### 6. Const Declaration

```go
// const MaxAmount = 1000000
Const().Id("MaxAmount").Op("=").Lit(1000000)
```

## Control Flow

### 7. If Statement

```go
// if err != nil {
//     return err
// }
If(Err().Op("!=").Nil()).Block(
    Return(Err()),
)
```

### 8. If with Assignment

```go
// if err := doSomething(); err != nil {
//     return err
// }
If(
    Err().Op(":=").Id("doSomething").Call(),
    Err().Op("!=").Nil(),
).Block(
    Return(Err()),
)
```

### 9. For Loop

```go
// for i, item := range items {
//     process(item)
// }
For(List(Id("i"), Id("item")).Op(":=").Range().Id("items")).Block(
    Id("process").Call(Id("item")),
)
```

### 10. Switch Statement

```go
// switch value {
// case 1:
//     return "one"
// default:
//     return "unknown"
// }
Switch(Id("value")).Block(
    Case(Lit(1)).Block(
        Return(Lit("one")),
    ),
    Default().Block(
        Return(Lit("unknown")),
    ),
)
```

## Common Patterns

### 11. Error Wrapping

```go
// return fmt.Errorf("failed to parse: %w", err)
Return(
    Qual("fmt", "Errorf").Call(
        Lit("failed to parse: %w"),
        Err(),
    ),
)
```

### 12. Multiple Return Values

```go
// return result, nil
Return(Id("result"), Nil())

// return nil, err
Return(Nil(), Err())
```

### 13. Slice Declaration

```go
// accounts := make([]solana.PublicKey, 0, 10)
Id("accounts").Op(":=").Make(
    Index().Qual("github.com/gagliardetto/solana-go", "PublicKey"),
    Lit(0),
    Lit(10),
)
```

### 14. Append to Slice

```go
// accounts = append(accounts, key)
Id("accounts").Op("=").Append(Id("accounts"), Id("key"))
```

### 15. Map Declaration

```go
// tags := map[string]string{"json": "amount"}
Id("tags").Op(":=").Map(String()).String().Values(
    Dict{
        Lit("json"): Lit("amount"),
    },
)
```

## Advanced Patterns

### 16. Function with Multiple Params

```go
// func NewInstruction(
//     amount uint64,
//     account solana.PublicKey,
// ) (solana.Instruction, error)
Func().Id("NewInstruction").
    Params(
        Id("amount").Uint64(),
        Id("account").Qual("github.com/gagliardetto/solana-go", "PublicKey"),
    ).
    Params(
        Qual("github.com/gagliardetto/solana-go", "Instruction"),
        Error(),
    ).
    Block(...)
```

### 17. Function with ParamsFunc (Dynamic)

```go
Func().Id("NewInstruction").
    ParamsFunc(func(g *Group) {
        for _, arg := range args {
            g.Id(arg.Name).Uint64()
        }
    }).
    Params(Error()).
    Block(...)
```

### 18. Struct with StructFunc (Dynamic)

```go
Type().Id("Config").StructFunc(func(g *Group) {
    for _, field := range fields {
        g.Id(field.Name).String()
    }
})
```

### 19. Block with BlockFunc (Dynamic)

```go
Func().Id("Process").Params().Block(
    BlockFunc(func(g *Group) {
        for i := 0; i < 5; i++ {
            g.Id("doSomething").Call(Lit(i))
        }
    }),
)
```

### 20. Comment and Line

```go
Comment("This is a comment")
Line()
Comment("Another comment")
```

### 21. Multi-line Comments

```go
Comment("Function Description:")
Comment("- Does something")
Comment("- Returns error if failed")
Line()
Func().Id("DoSomething")...
```

## Anchor/Solana Specific

### 22. Discriminator Array

```go
// var SwapDiscriminator = [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
Var().Id("SwapDiscriminator").Op("=").Index(Lit(8)).Byte().Values(
    Lit(0x01), Lit(0x02), Lit(0x03), Lit(0x04),
    Lit(0x05), Lit(0x06), Lit(0x07), Lit(0x08),
)
```

### 23. Borsh Encoding

```go
// enc := bin.NewBorshEncoder(buf)
// if err := enc.Encode(value); err != nil {
//     return err
// }
Id("enc").Op(":=").Qual("github.com/gagliardetto/binary", "NewBorshEncoder").Call(Id("buf"))
If(
    Err().Op(":=").Id("enc").Dot("Encode").Call(Id("value")),
    Err().Op("!=").Nil(),
).Block(
    Return(Err()),
)
```

### 24. Borsh Decoding

```go
// dec := bin.NewBorshDecoder(data)
// if err := dec.Decode(&obj); err != nil {
//     return err
// }
Id("dec").Op(":=").Qual("github.com/gagliardetto/binary", "NewBorshDecoder").Call(Id("data"))
If(
    Err().Op(":=").Id("dec").Dot("Decode").Call(Op("&").Id("obj")),
    Err().Op("!=").Nil(),
).Block(
    Return(Err()),
)
```

### 25. Account Meta

```go
// solana.NewAccountMeta(pubkey, writable, signer)
Qual("github.com/gagliardetto/solana-go", "NewAccountMeta").Call(
    Id("pubkey"),
    Lit(true),  // writable
    Lit(false), // signer
)
```

### 26. New Instruction

```go
// solana.NewInstruction(
//     programID,
//     accounts,
//     data,
// )
Qual("github.com/gagliardetto/solana-go", "NewInstruction").Call(
    Id("programID"),
    Id("accounts"),
    Id("data"),
)
```

### 27. Option Type Handling

```go
// if value != nil {
//     if err := enc.WriteOption(true); err != nil {
//         return err
//     }
//     if err := enc.Encode(*value); err != nil {
//         return err
//     }
// } else {
//     if err := enc.WriteOption(false); err != nil {
//         return err
//     }
// }
If(Id("value").Op("!=").Nil()).Block(
    If(
        Err().Op(":=").Id("enc").Dot("WriteOption").Call(True()),
        Err().Op("!=").Nil(),
    ).Block(Return(Err())),
    If(
        Err().Op(":=").Id("enc").Dot("Encode").Call(Op("*").Id("value")),
        Err().Op("!=").Nil(),
    ).Block(Return(Err())),
).Else().Block(
    If(
        Err().Op(":=").Id("enc").Dot("WriteOption").Call(False()),
        Err().Op("!=").Nil(),
    ).Block(Return(Err())),
)
```

## Utilities

### 28. Do Statement (Conditional Code)

```go
// Conditionally add code
Id("value").Op("=").Do(func(s *Statement) {
    if condition {
        s.Lit(100)
    } else {
        s.Lit(200)
    }
})
```

### 29. DoGroup (Group of Statements)

```go
Params(
    DoGroup(func(g *Group) {
        g.Id("param1").String()
        if includeParam2 {
            g.Id("param2").Int()
        }
    }),
)
```

### 30. Empty Statement

```go
// Start with empty and build up
code := Empty()
code.Func().Id("Test")...
code.Line()
code.Func().Id("Another")...
```

## File Operations

### 31. Create File with Package

```go
file := NewFile("mypackage")
file.HeaderComment("Code generated. DO NOT EDIT.")
file.Add(...)
```

### 32. Add Imports

```go
file := NewFile("mypackage")
file.ImportName("github.com/gagliardetto/solana-go", "solana")
file.ImportAlias("github.com/gagliardetto/binary", "bin")
```

### 33. Render to String

```go
code := file.GoString()
```

### 34. Render to Writer

```go
err := file.Render(os.Stdout)
```

## Best Practices

### 35. Use Qual for External Packages

```go
// Good
Qual("github.com/gagliardetto/solana-go", "PublicKey")

// Bad (won't add import)
Id("solana").Dot("PublicKey")
```

### 36. Use Line() for Spacing

```go
Func().Id("First")...
Line()
Line()  // Extra spacing
Func().Id("Second")...
```

### 37. Group Related Code

```go
File.Block(
    Comment("Constants"),
    Const().Id("Version").Op("=").Lit("1.0.0"),
    Const().Id("Name").Op("=").Lit("MyProgram"),
).Line().Line()

File.Block(
    Comment("Variables"),
    Var().Id("ProgramID").Qual("github.com/gagliardetto/solana-go", "PublicKey"),
)
```

### 38. Use Dict for Map Literals

```go
Map(String()).String().Values(
    Dict{
        Lit("json"):  Lit("amount"),
        Lit("borsh"): Lit("amount"),
    },
)
```

### 39. Use List for Multiple Values

```go
// return value1, value2, nil
Return(List(Id("value1"), Id("value2"), Nil()))
```

### 40. Format Check

```go
// Always format check generated code
code := file.GoString()
formatted, err := format.Source([]byte(code))
if err != nil {
    log.Fatal("Generated code has syntax error:", err)
}
```

## Complete Example

```go
package main

import (
    . "github.com/dave/jennifer/jen"
    "os"
)

func main() {
    file := NewFile("generated")
    
    // Add header
    file.HeaderComment("Code generated by go-carbon. DO NOT EDIT.")
    file.Line()
    
    // Add imports
    file.ImportName("github.com/gagliardetto/solana-go", "solana")
    
    // Add constant
    file.Const().Id("ProgramID").Op("=").Lit("TokenSwap...")
    file.Line()
    
    // Add function
    file.Func().Id("NewSwapInstruction").
        Params(
            Id("amount").Uint64(),
            Id("account").Qual("github.com/gagliardetto/solana-go", "PublicKey"),
        ).
        Params(
            Qual("github.com/gagliardetto/solana-go", "Instruction"),
            Error(),
        ).
        Block(
            Comment("Validate account"),
            If(Id("account").Dot("IsZero").Call()).Block(
                Return(Nil(), Qual("fmt", "Errorf").Call(Lit("account required"))),
            ),
            Line(),
            Comment("Build instruction"),
            Return(
                Qual("github.com/gagliardetto/solana-go", "NewInstruction").Call(
                    Id("ProgramID"),
                    Nil(),
                    Nil(),
                ),
                Nil(),
            ),
        )
    
    // Render
    file.Render(os.Stdout)
}
```

## Tips

1. **Start Simple**: Begin with basic structures, add complexity gradually
2. **Test Often**: Generate and compile frequently to catch errors
3. **Use GoString()**: Debug by printing `code.GoString()`
4. **Read Jennifer Docs**: https://pkg.go.dev/github.com/dave/jennifer/jen
5. **Study Examples**: Look at anchor-go generator code for patterns
6. **Format Check**: Always validate generated code with `go/format`

---

**Happy Code Generating! 🚀**
