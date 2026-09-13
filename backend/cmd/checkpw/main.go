package main

import (
	"fmt"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
)

func main() {
	hash := "$2b$12$tJAUOdCuGadmHyNlXjE9gOx2IuQhVG8cU8z.Cg8j49EybLUt2jOhe"
	pw := "SuperAdmin#2026"

	fmt.Println("CheckPassword result:", security.CheckPassword(pw, hash))
}
