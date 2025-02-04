package main

import (
	"fmt"
	"os"

	"github.com/Vaelatern/nomad-declarative/internal/templating"
)

func main() {
	fmt.Println(templating.ListTemplates(os.DirFS("./testpacks")))
}
