package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnzipFile extracts a ZIP file specified by zipPath to the destination directory destPath.
func UnzipFile(zipPath string, destPath string) error {
	err := os.MkdirAll(destPath, 0755)
	if err != nil {
		return err
	}
	if strings.HasSuffix(zipPath, ".tgz") {
		unzipTgz := func() error {
			// 打开 tgz 文件
			file, err := os.Open(zipPath)
			if err != nil {
				return err
			}
			defer file.Close()

			// 解压缩 gzip
			gzReader, err := gzip.NewReader(file)
			if err != nil {
				return err
			}
			defer gzReader.Close()

			// 解归档 tar
			tarReader := tar.NewReader(gzReader)

			for {
				header, err := tarReader.Next()
				if err == io.EOF {
					break // 结束循环
				}
				if err != nil {
					return err
				}

				// 构造目标文件路径
				targetPath := filepath.Join(destPath, header.Name)
				switch header.Typeflag {
				case tar.TypeDir:
					if err := os.MkdirAll(targetPath, 0755); err != nil {
						return err
					}
				case tar.TypeReg:
					if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
						return err
					}
					outFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
					if err != nil {
						return err
					}
					defer outFile.Close()
					if _, err := io.Copy(outFile, tarReader); err != nil {
						return err
					}
				}
			}

			return nil
		}
		return unzipTgz()
	}

	// Open the ZIP file
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zipReader.Close()

	// Ensure the destination directory exists
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Iterate over each file in the archive
	for _, file := range zipReader.File {
		filePath := filepath.Join(destPath, file.Name)

		// // Prevent directory traversal attacks by validating the path
		// if !filepath.HasPrefix(filePath, filepath.Clean(destPath)+string(os.PathSeparator)) {
		// 	return fmt.Errorf("invalid file path: %s", filePath)
		// }

		if file.FileInfo().IsDir() {
			// Create directories
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Extract file
			// extractFile extracts a single file from the archive.
			extractFile := func(file *zip.File, filePath string) error {
				// Open the file inside the ZIP archive
				fileReader, err := file.Open()
				if err != nil {
					return fmt.Errorf("failed to open file in zip archive: %w", err)
				}
				defer fileReader.Close()

				// Create the destination file
				outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
				if err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}
				defer outFile.Close()

				// Copy file contents
				if _, err := io.Copy(outFile, fileReader); err != nil {
					return fmt.Errorf("failed to copy file contents: %w", err)
				}

				return nil
			}
			if err := extractFile(file, filePath); err != nil {
				return err
			}
		}
	}

	return nil
}

// func UnzipFile(zipPath string, destPath string) error {
// 	_, err := ModRunCmd.NewBuilder("unzip", zipPath, "-d", destPath)
// 	return err
// }

// zipDirectory 压缩目录为 ZIP 文件
func ZipDirectory(sourceDir, destZip string) error {
	// 创建 ZIP 文件
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// 创建 ZIP writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历目录并添加文件到 ZIP
	fmt.Println("walkdir", sourceDir)
	return filepath.Walk(filepath.Clean(sourceDir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error1:", err)
			return err
		}

		// 获取文件的相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			fmt.Println("Error2:", err)
			return err
		}

		// 如果是目录，添加目录信息
		if info.IsDir() {
			if relPath != "." { // 根目录不需要添加到 ZIP
				_, err = zipWriter.Create(relPath + "/")
			}
			if err != nil {
				fmt.Println("Error3:", err)
			}
			return err
		}

		// 如果是文件，写入到 ZIP
		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			fmt.Println("Error4:", err)
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			fmt.Println("Error5:", err)
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFileWriter, file)
		if err != nil {
			fmt.Println("Error6:", err)
		}
		return err
	})
}
