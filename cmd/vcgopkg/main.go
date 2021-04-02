package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/otiai10/copy"
)

func main() {
	flag.Parse()
	inputPath := flag.Arg(0)
	fmt.Println(inputPath)

	inputPathStat, err := os.Stat(inputPath)
	if err != nil {
		log.Printf("Error getting stat for '%s'. File or path may not exist? Original error: '%s'", inputPath, err)
		os.Exit(1)
		return
	}

	log.Print("Reading go files in: " + inputPath)

	mainFiles := []string{}
	if inputPathStat.IsDir() {
		log.Printf("'%s' input is dir", inputPath)
		err := filepath.Walk(inputPath,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return nil
				}

				parsedPackages, err := parser.ParseDir(
					token.NewFileSet(),
					path,
					nil,
					parser.ParseComments,
				)

				if err != nil {
					panic(err)
				}

				for _, pkg := range parsedPackages {
					for _, file := range pkg.Files {
						for _, decl := range file.Decls {
							if ast.FilterDecl(decl, func(name string) bool { return name == "main" }) {
								mainFiles = append(mainFiles, path+"/"+file.Name.Name+".go")
							}
						}
					}
				}
				return nil
			})
		if err != nil {
			log.Fatal(err)
		}
	} else if inputPathStat.Mode().Perm().IsRegular() {
		log.Printf("'%s' input is a file", inputPath)
		parsedFile, err := parser.ParseFile(token.NewFileSet(), inputPath, nil, parser.ParseComments)

		if err != nil {
			panic(err)
		}

		for _, decl := range parsedFile.Decls {
			if ast.FilterDecl(decl, func(name string) bool { return name == "main" }) {
				mainFiles = append(mainFiles, inputPath)
			}
		}
	} else {
		log.Fatalf("'%s' does not exist", inputPath)
		os.Exit(1)
	}
	if len(mainFiles) == 0 {
		log.Fatalf("No main files found in %s", inputPath)
	}
	spew.Dump(mainFiles)

	for _, mainFile := range mainFiles {
		goModPath := ""
		parentDir := path.Dir(mainFile)
		for {
			goModStat, _ := os.Stat(parentDir + "/go.mod")

			if goModStat == nil {
				if parentDir != "" {
					parentDir = path.Dir(parentDir)
					continue
				} else {
					break
				}
			}

			goModPath = parentDir + "/go.mod"
			break
		}

		println(goModPath)
		tempWorkDir, err := os.MkdirTemp("", "vcgopkg")
		println(tempWorkDir)
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(tempWorkDir) // clean up

		copyDir := tempWorkDir + "/" + filepath.Base(filepath.Dir(goModPath))
		copy.Copy(parentDir, copyDir)

		cmd := exec.Command("ls", "-lah", copyDir)
		cmdOut, _ := cmd.Output()
		println(string(cmdOut))

		cmd = exec.Command("go", "mod", "vendor")
		cmd.Dir = copyDir
		cmdOut, _ = cmd.Output()
		println(string(cmdOut))

		json := []byte("{\"MainFile\": \"cmd/go-detailedreport-to-csv\"}")
		ioutil.WriteFile(copyDir+"/veracode.json", json, 0644)

		baseDir := filepath.Base(filepath.Dir(goModPath))
		cmd = exec.Command("zip", "-r", baseDir+".zip", baseDir)
		cmd.Dir = tempWorkDir
		cmdOut, _ = cmd.Output()
		println(string(cmdOut))

		veracodeDir := parentDir + "/veracode"
		os.Mkdir(veracodeDir, 0777)

		cmd = exec.Command("mv", baseDir+".zip", veracodeDir)
		cmd.Dir = tempWorkDir
		cmdOut, _ = cmd.Output()
		println(baseDir + ".zip")
		println(string(cmdOut))

		cmd = exec.Command("ls", "-lah", tempWorkDir)
		cmdOut, _ = cmd.Output()
		println(string(cmdOut))
	}
}
