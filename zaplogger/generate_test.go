package zaplogger

import (
	"testing"

	"goa.design/goa/codegen"
	"goa.design/goa/codegen/generator"
	"goa.design/goa/eval"
	"goa.design/goa/expr"
	httpcodegen "goa.design/goa/http/codegen"
	"goa.design/plugins/zaplogger/testdata"
)

func TestGenerate(t *testing.T) {

	httpcodegen.RunHTTPDSL(t, testdata.SimpleServiceDSL)

	roots := []eval.Root{expr.Root}
	files := generateFiles(t, roots)
	newFiles, err := Generate("", roots, files)

	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	newFilesCount := len(newFiles) - len(files)

	if newFilesCount != 1 {
		t.Errorf("invalid code: number of new files expected %d, got %d", 1, newFilesCount)
	}
}

func generateFiles(t *testing.T, roots []eval.Root) []*codegen.File {

	files, err := generator.Service("", roots)
	if err != nil {
		t.Fatalf("error in code generation: %v", err)
	}
	httpFiles, err := generator.Transport("", roots)
	if err != nil {
		t.Fatalf("error in HTTP code generation: %v", err)
	}
	files = append(files, httpFiles...)
	return files
}
