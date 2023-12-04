package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"strings"
)

func UntarFile(tarFile, dstFolder string) error {
	reader, err := os.Open(tarFile)
	if err != nil {
		return err
	}

	defer reader.Close()
	return Untar(reader, NewFolderRouter(dstFolder))
}

func Untar(source io.Reader, router Router) error {
	tarReader := tar.NewReader(source)

	for {
		header, err := tarReader.Next()
		fmt.Printf("tarReader.next Debug: %+v\n", header)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := router.ExtractPath(header)
		if path == "" {
			continue
		}

		// Prevent malicous directory traversals.
		// https://cwe.mitre.org/data/definitions/22.html
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("file in tarball contains a directory traversal component (..): %v", header.Name)
		}

		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		fmt.Printf("Untar Debug: opening file %s\n", path)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}

		fmt.Printf("Untar Debug: copying file %v\n", file)
		if _, err = io.Copy(file, tarReader); err != nil {
			// In Go, "defer" will be executed when the function returns, not at the end of the loop. File descriptor limits could be reached if we don't close the file here.
			file.Close()
			return err
		}
		file.Close()
	}
	return nil
}
