package main

import (
    "fmt"
    goseams "groundseg/internal/seams"
)

type outer struct {
    inner inner
}

type inner struct {
    F func()
    G func()
}

func main() {
    base := outer{inner: inner{F: func() { fmt.Println("base") }}}
    override := outer{inner: inner{F: func() { fmt.Println("override") }}}
    got := goseams.Merge(base, override)
    fmt.Printf("%p %v\n", got.inner.F, got.inner.G == nil)
    if got.inner.F != nil {
        got.inner.F()
    }
}
